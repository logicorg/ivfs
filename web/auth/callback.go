package auth

import (
	"errors"
	"log"
	"net/http"
	"slices"
	"time"

	"github.com/logicorg/ivfs/web/api/gateway"
	"golang.org/x/oauth2"
)

const startAgain = "; start again at " + LoginPath

// callback checks the provider's answer against the sign-in this browser
// began, redeems the code for an ID token, verifies it, and signs in the
// user it names, setting them up on their first sign-in. ivfs keeps none of
// the provider's tokens.
func (h *Handlers) callback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	st, ok := h.signer.loginStateFrom(r, time.Now())
	http.SetCookie(w, clearLoginState())
	if !ok {
		gateway.WriteProblem(w, http.StatusBadRequest, "this sign-in expired or began in another browser"+startAgain)
		return
	}
	query := r.URL.Query()
	// Checked before anything else in the answer, so another site cannot
	// send this browser here with a code or an error of its choosing.
	if query.Get("state") != st.State {
		gateway.WriteProblem(w, http.StatusBadRequest, "the provider's answer is for another sign-in"+startAgain)
		return
	}
	if refusal := query.Get("error"); refusal != "" {
		gateway.WriteProblem(w, http.StatusUnauthorized, "the identity provider refused the sign-in: "+refusal+startAgain)
		return
	}

	idp, err := h.idp.discover(ctx)
	if err != nil {
		log.Printf("Sign-in: %v", err)
		gateway.WriteProblem(w, http.StatusBadGateway, "the identity provider is unreachable; try again shortly")
		return
	}
	token, err := idp.oauth2.Exchange(h.idp.clientContext(ctx), query.Get("code"), oauth2.VerifierOption(st.Verifier))
	if err != nil {
		log.Printf("Sign-in: redeem code: %v", err)
		var refused *oauth2.RetrieveError
		if errors.As(err, &refused) {
			gateway.WriteProblem(w, http.StatusUnauthorized, "the identity provider did not accept the sign-in"+startAgain)
			return
		}
		gateway.WriteProblem(w, http.StatusBadGateway, "the identity provider is unreachable; try again shortly")
		return
	}
	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		gateway.WriteProblem(w, http.StatusBadGateway, "the identity provider returned no ID token; check that the client has the openid scope")
		return
	}
	idToken, err := idp.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		log.Printf("Sign-in: verify ID token: %v", err)
		gateway.WriteProblem(w, http.StatusUnauthorized, "the ID token is not valid"+startAgain)
		return
	}
	if idToken.Nonce != st.Nonce {
		gateway.WriteProblem(w, http.StatusUnauthorized, "the ID token is for another sign-in"+startAgain)
		return
	}
	if h.idp.Group != "" {
		var claims struct {
			Groups []string `json:"groups"`
		}
		if err := idToken.Claims(&claims); err != nil || !slices.Contains(claims.Groups, h.idp.Group) {
			gateway.WriteProblem(w, http.StatusForbidden, "you are not in the group "+h.idp.Group+"; ask an administrator to add you")
			return
		}
	}

	uid, err := h.ensure(ctx, idToken.Issuer, idToken.Subject)
	if err != nil {
		log.Printf("Sign-in: ensure user: %v", err)
		gateway.WriteProblem(w, http.StatusServiceUnavailable, "your account could not be set up yet"+startAgain+" to finish it")
		return
	}
	http.SetCookie(w, h.signer.Session(uid, time.Now()))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
