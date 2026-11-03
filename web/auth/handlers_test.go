package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

// signIn is a gateway's sign-in endpoints, against a provider.
type signIn struct {
	router http.Handler
	signer *auth.Signer
	idp    *auth.Provider
}

func newSignIn(t *testing.T, idp *fakeProvider, group string, ensure auth.EnsureUser) *signIn {
	t.Helper()
	if ensure == nil {
		ensure = func(context.Context, string, string) (uuid.UUID, error) {
			t.Error("ensure ran")
			return uuid.Nil, nil
		}
	}
	s := &signIn{
		signer: newSigner(t, keyA),
		idp: &auth.Provider{
			Issuer: idp.issuer(), ClientID: clientID, ClientSecret: "secret", PublicURL: publicURL, Group: group,
			HTTPClient: idp.server.Client(),
		},
	}
	router := chi.NewMux()
	auth.NewHandlers(s.signer, s.idp, ensure).Register(router)
	s.router = router
	return s
}

func (s *signIn) serve(method, target string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, publicURL+target, nil)
	for _, c := range cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	s.router.ServeHTTP(w, r)
	return w
}

// cookie returns the cookie named name that w sets, or nil.
func cookie(w *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range w.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestHandlers_Register(t *testing.T) {
	s := newSignIn(t, newFakeProvider(t, false), "", nil)

	tests := map[string]struct {
		method, path string
		wantStatus   int
	}{
		"logging out by link": {method: http.MethodGet, path: auth.LogoutPath, wantStatus: http.StatusMethodNotAllowed},
		"posting a login":     {method: http.MethodPost, path: auth.LoginPath, wantStatus: http.StatusMethodNotAllowed},
		"posting a callback":  {method: http.MethodPost, path: auth.CallbackPath, wantStatus: http.StatusMethodNotAllowed},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.wantStatus, s.serve(tc.method, tc.path).Code)
		})
	}
}
