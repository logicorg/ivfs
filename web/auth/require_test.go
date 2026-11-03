package auth_test

import (
	"net/http"
	"net/http/httptest"
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
		// wantUser is the user the next handler acts as; nil means it is
		// not reached.
		wantUser *uuid.UUID
	}{
		"an API request with a session": {
			path: "/api/currencies", cookie: session, wantStatus: http.StatusOK, wantUser: &uid,
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
			next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				reached = true
				got, _ = tenancy.UserFrom(r.Context())
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
			if tc.wantStatus == http.StatusUnauthorized {
				require.Equal(t, "application/problem+json", w.Header().Get("Content-Type"))
				require.Contains(t, w.Body.String(), "sign in at /auth/login")
			}
		})
	}
}
