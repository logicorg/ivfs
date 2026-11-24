package auth_test

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

func TestRequire(t *testing.T) {
	uid := uuid.Must(uuid.NewV7())
	signer := newSigner(t, keyA)
	session := signer.Session(uid, time.Now())
	expired := signer.Session(uid, time.Now().Add(-auth.SessionLifetime))

	tests := map[string]struct {
		path         string
		cookie       *http.Cookie
		wantStatus   int
		wantLocation string
		// nextStatus is what the next handler answers with, 200 when zero.
		nextStatus int
		// wantUser is the user the next handler acts as; nil means it is
		// not reached.
		wantUser *uuid.UUID
		// wantSignedOut means the response clears the session cookie.
		wantSignedOut bool
	}{
		"an API request with a session": {
			path: "/api/currencies", cookie: session, wantStatus: http.StatusOK, wantUser: &uid,
		},
		// The session's user is gone, as after the registry is recreated, so
		// the session is cleared rather than failing every request until it
		// expires.
		"an API request whose session the API refuses": {
			path: "/api/currencies", cookie: session, nextStatus: http.StatusUnauthorized,
			wantStatus: http.StatusUnauthorized, wantUser: &uid, wantSignedOut: true,
		},
		"an API request the API refuses for another reason": {
			path: "/api/currencies", cookie: session, nextStatus: http.StatusNotFound,
			wantStatus: http.StatusNotFound, wantUser: &uid,
		},
		"a page with a session": {
			path: "/accounts", cookie: session, wantStatus: http.StatusOK, wantUser: &uid,
		},
		"an API request without a session": {
			path: "/api/currencies", wantStatus: http.StatusUnauthorized,
		},
		"an API request with an expired session": {
			path: "/api/currencies", cookie: expired, wantStatus: http.StatusUnauthorized,
		},
		"the API document without a session": {
			path: "/api/openapi.json", wantStatus: http.StatusUnauthorized,
		},
		"a page without a session": {
			path: "/accounts", wantStatus: http.StatusSeeOther, wantLocation: auth.LoginPath,
		},
		"a lookalike of the health check": {
			path: "/api/health/x", wantStatus: http.StatusUnauthorized,
		},
		"the health check without a session": {
			path: "/api/health", wantStatus: http.StatusOK, wantUser: &uuid.Nil,
		},
		"signing in without a session": {
			path: "/auth/login", wantStatus: http.StatusOK, wantUser: &uuid.Nil,
		},
		// A public path never acts as anyone, so a handler there cannot
		// open a user's data even when the browser has signed in.
		"signing in with a session": {
			path: "/auth/callback", cookie: session, wantStatus: http.StatusOK, wantUser: &uuid.Nil,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var reached bool
			var got uuid.UUID
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				reached = true
				got, _ = tenancy.UserFrom(r.Context())
				if tc.nextStatus != 0 {
					w.WriteHeader(tc.nextStatus)
				}
			})
			r := httptest.NewRequest(http.MethodGet, tc.path, nil)
			if tc.cookie != nil {
				r.AddCookie(tc.cookie)
			}
			w := httptest.NewRecorder()

			auth.Require(signer)(next).ServeHTTP(w, r)

			require.Equal(t, tc.wantStatus, w.Code, w.Body.String())
			require.Equal(t, tc.wantLocation, w.Header().Get("Location"))
			require.Equal(t, tc.wantUser != nil, reached)
			if tc.wantUser != nil {
				require.Equal(t, *tc.wantUser, got)
			}
			require.Equal(t, tc.wantSignedOut, slices.ContainsFunc(w.Result().Cookies(), func(c *http.Cookie) bool {
				return c.Name == auth.SessionCookie && c.MaxAge < 0
			}))
			if tc.wantStatus == http.StatusUnauthorized && tc.nextStatus == 0 {
				require.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
				require.Contains(t, w.Body.String(), "sign in at /auth/login")
			}
		})
	}
}
