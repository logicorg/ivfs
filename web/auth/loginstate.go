package auth

import (
	"encoding/json"
	"net/http"
	"time"
)

const (
	// loginCookie keeps a sign-in's secrets while the browser is at the
	// provider. It is sent only to the callback, which a __Host- cookie,
	// bound to /, could not be.
	loginCookie   = "__Secure-login"
	loginLifetime = 10 * time.Minute
	loginPurpose  = "login"
)

// loginState is what the callback checks the provider's answer against:
// state ties the answer to this browser, nonce ties the ID token to this
// sign-in, and verifier proves the code is redeemed by who asked for it.
type loginState struct {
	State    string `json:"state"`
	Nonce    string `json:"nonce"`
	Verifier string `json:"verifier"`
	// Exp is signed, unlike the cookie's own expiry, which the browser
	// enforces only if it wants to.
	Exp int64 `json:"exp"`
}

// loginStateCookie returns the cookie that keeps st until loginLifetime from
// now.
func (s *Signer) loginStateCookie(st loginState, now time.Time) (*http.Cookie, error) {
	st.Exp = now.Add(loginLifetime).Unix()
	payload, err := json.Marshal(st)
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     loginCookie,
		Value:    s.seal(loginPurpose, payload),
		Path:     CallbackPath,
		MaxAge:   int(loginLifetime.Seconds()),
		Secure:   true,
		HttpOnly: true,
		// Lax reaches the callback, which is a top-level navigation back
		// from the provider.
		SameSite: http.SameSiteLaxMode,
	}, nil
}

// loginStateFrom returns the sign-in r's login cookie keeps, unless it is
// missing, forged, or expired at now.
func (s *Signer) loginStateFrom(r *http.Request, now time.Time) (loginState, bool) {
	cookie, err := r.Cookie(loginCookie)
	if err != nil {
		return loginState{}, false
	}
	payload, ok := s.open(loginPurpose, cookie.Value)
	if !ok {
		return loginState{}, false
	}
	var st loginState
	if err := json.Unmarshal(payload, &st); err != nil || !now.Before(time.Unix(st.Exp, 0)) {
		return loginState{}, false
	}
	return st, true
}

// clearLoginState returns the cookie that ends a sign-in, whichever way it
// went, so a finished one leaves nothing to replay.
func clearLoginState() *http.Cookie {
	return &http.Cookie{
		Name:     loginCookie,
		Path:     CallbackPath,
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}
