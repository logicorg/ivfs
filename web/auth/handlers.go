package auth

import (
	"context"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// The sign-in endpoints. They need no session, as Require lets every path
// under /auth/ through.
const (
	// LoginPath is where a browser without a session goes to sign in.
	LoginPath = "/auth/login"
	// CallbackPath is where the provider sends the browser back.
	CallbackPath = "/auth/callback"
	LogoutPath   = "/auth/logout"
)

// EnsureUser returns the user an identity belongs to, registering it and
// making its database ready on its first sign-in.
type EnsureUser func(ctx context.Context, issuer, subject string) (uuid.UUID, error)

// Handlers sign users in with idp and out again, keeping who they are in
// sessions signer signs.
type Handlers struct {
	signer *Signer
	idp    *Provider
	ensure EnsureUser
}

// NewHandlers returns the sign-in endpoints.
func NewHandlers(signer *Signer, idp *Provider, ensure EnsureUser) *Handlers {
	return &Handlers{signer: signer, idp: idp, ensure: ensure}
}

// Register adds the sign-in endpoints to router.
func (h *Handlers) Register(router chi.Router) {
	router.Get(LoginPath, h.login)
	router.Get(CallbackPath, h.callback)
	// Signing out changes state, so it is a POST, which a link or an image
	// on another page cannot send.
	router.Post(LogoutPath, h.logout)
}
