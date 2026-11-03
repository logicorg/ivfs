package auth

import (
	"crypto/rand"
	"log"
	"net/http"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/logicorg/ivfs/web/api/gateway"
	"golang.org/x/oauth2"
)

// login sends the browser to the provider, with the authorization code flow
// and PKCE, and keeps what the callback checks the answer against.
func (h *Handlers) login(w http.ResponseWriter, r *http.Request) {
	idp, err := h.idp.discover(r.Context())
	if err != nil {
		log.Printf("Sign-in: %v", err)
		gateway.WriteProblem(w, http.StatusBadGateway, "the identity provider is unreachable; try again shortly")
		return
	}

	st := loginState{State: rand.Text(), Nonce: rand.Text(), Verifier: oauth2.GenerateVerifier()}
	cookie, err := h.signer.loginStateCookie(st, time.Now())
	if err != nil {
		log.Printf("Sign-in: seal login state: %v", err)
		gateway.WriteProblem(w, http.StatusInternalServerError, "the sign-in could not start; the gateway's log has the details")
		return
	}
	http.SetCookie(w, cookie)
	http.Redirect(w, r, idp.oauth2.AuthCodeURL(st.State, oidc.Nonce(st.Nonce), oauth2.S256ChallengeOption(st.Verifier)), http.StatusFound)
}
