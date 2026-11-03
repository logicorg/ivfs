package auth

import (
	"log"
	"net/http"
	"net/url"
)

// logout ends the session and sends the browser to end its session at the
// provider too, at its end_session_endpoint or else at its LogoutURL:
// otherwise the next visit is signed straight back in by the provider's own
// session.
func (h *Handlers) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, ClearSession())

	target := "/"
	if h.idp.LogoutURL != "" {
		target = h.idp.LogoutURL
	}
	idp, err := h.idp.discover(r.Context())
	switch {
	case err != nil:
		log.Printf("Sign-out: %v; sending the browser to %s", err, target)
	case idp.endSession != "":
		endSession, err := url.Parse(idp.endSession)
		if err != nil {
			log.Printf("Sign-out: end_session_endpoint %q: %v; sending the browser to %s", idp.endSession, err, target)
			break
		}
		query := endSession.Query()
		// No id_token_hint: the gateway kept no ID token to send.
		query.Set("client_id", h.idp.ClientID)
		query.Set("post_logout_redirect_uri", h.idp.PublicURL+"/")
		endSession.RawQuery = query.Encode()
		target = endSession.String()
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}
