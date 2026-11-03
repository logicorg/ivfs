package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

func TestHandlers_login(t *testing.T) {
	idp := newFakeProvider(t, false)

	tests := map[string]struct {
		group      string
		wantScopes string
	}{
		"anyone the provider knows": {wantScopes: "openid"},
		"only a group":              {group: "ivfs-users", wantScopes: "openid groups"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSignIn(t, idp, tc.group, nil)

			w := s.serve(http.MethodGet, auth.LoginPath)

			require.Equal(t, http.StatusFound, w.Code, w.Body.String())
			to, err := url.Parse(w.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, idp.issuer()+"/authorize", to.Scheme+"://"+to.Host+to.Path)
			q := to.Query()
			require.Equal(t, "code", q.Get("response_type"))
			require.Equal(t, clientID, q.Get("client_id"))
			require.Equal(t, publicURL+auth.CallbackPath, q.Get("redirect_uri"))
			require.Equal(t, tc.wantScopes, q.Get("scope"))
			require.Equal(t, "S256", q.Get("code_challenge_method"))
			for _, param := range []string{"state", "nonce", "code_challenge"} {
				require.GreaterOrEqual(t, len(q.Get(param)), 26, "%s carries at least 128 random bits", param)
			}

			state := cookie(w, "__Secure-login")
			require.NotNil(t, state)
			require.Equal(t, auth.CallbackPath, state.Path)
			require.Equal(t, 600, state.MaxAge)
			require.True(t, state.Secure)
			require.True(t, state.HttpOnly)
			require.Equal(t, http.SameSiteLaxMode, state.SameSite)
		})
	}
}

// TestHandlers_login_Fresh checks that each sign-in has its own secrets, so
// one answer cannot finish another.
func TestHandlers_login_Fresh(t *testing.T) {
	s := newSignIn(t, newFakeProvider(t, false), "", nil)

	first, err := url.Parse(s.serve(http.MethodGet, auth.LoginPath).Header().Get("Location"))
	require.NoError(t, err)
	second, err := url.Parse(s.serve(http.MethodGet, auth.LoginPath).Header().Get("Location"))
	require.NoError(t, err)

	for _, param := range []string{"state", "nonce", "code_challenge"} {
		require.NotEqual(t, first.Query().Get(param), second.Query().Get(param), param)
	}
}

// TestHandlers_login_ProviderDown checks that a provider that is down fails
// the sign-in, and that the next one discovers it once it is back.
func TestHandlers_login_ProviderDown(t *testing.T) {
	idp := newFakeProvider(t, false)
	s := newSignIn(t, idp, "", nil)
	idp.down.Store(true)

	w := s.serve(http.MethodGet, auth.LoginPath)
	require.Equal(t, http.StatusBadGateway, w.Code)
	require.Contains(t, w.Body.String(), "unreachable")
	require.Nil(t, cookie(w, "__Secure-login"))

	idp.down.Store(false)
	require.Equal(t, http.StatusFound, s.serve(http.MethodGet, auth.LoginPath).Code)
}
