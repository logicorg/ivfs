package auth_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
)

func TestHandlers_logout(t *testing.T) {
	withEndSession := newFakeProvider(t, true)
	withoutEndSession := newFakeProvider(t, false)
	down := newFakeProvider(t, true)
	down.down.Store(true)

	const logoutURL = "https://auth.example/logout"

	tests := map[string]struct {
		idp       *fakeProvider
		logoutURL string
		// wantTarget is where the browser goes, without its query.
		wantTarget string
		wantQuery  url.Values
	}{
		"a provider with an end_session_endpoint": {
			idp: withEndSession, wantTarget: withEndSession.issuer() + "/logout",
			wantQuery: url.Values{"keep": {"1"}, "client_id": {clientID}, "post_logout_redirect_uri": {publicURL + "/"}},
		},
		"a provider without one": {
			idp: withoutEndSession, wantTarget: "/", wantQuery: url.Values{},
		},
		"a provider that is down": {
			idp: down, wantTarget: "/", wantQuery: url.Values{},
		},
		"a provider with an end_session_endpoint and a sign-out page": {
			idp: withEndSession, logoutURL: logoutURL, wantTarget: withEndSession.issuer() + "/logout",
			wantQuery: url.Values{"keep": {"1"}, "client_id": {clientID}, "post_logout_redirect_uri": {publicURL + "/"}},
		},
		"a provider with a sign-out page alone": {
			idp: withoutEndSession, logoutURL: logoutURL, wantTarget: logoutURL, wantQuery: url.Values{},
		},
		"a provider that is down, with a sign-out page": {
			idp: down, logoutURL: logoutURL, wantTarget: logoutURL, wantQuery: url.Values{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			s := newSignIn(t, tc.idp, "", nil)
			s.idp.LogoutURL = tc.logoutURL

			w := s.serve(http.MethodPost, auth.LogoutPath)

			require.Equal(t, http.StatusSeeOther, w.Code, w.Body.String())
			target, err := url.Parse(w.Header().Get("Location"))
			require.NoError(t, err)
			require.Equal(t, tc.wantQuery, target.Query())
			target.RawQuery = ""
			require.Equal(t, tc.wantTarget, target.String())

			cleared := cookie(w, auth.SessionCookie)
			require.NotNil(t, cleared, "the session ends whatever the provider does")
			require.Negative(t, cleared.MaxAge)
		})
	}
}
