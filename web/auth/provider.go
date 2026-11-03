package auth

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Provider is the OpenID Connect provider users sign in with (ADR 0013), and
// how the gateway is registered with it.
type Provider struct {
	// Issuer is the provider's issuer URL, where its discovery document is.
	Issuer       string
	ClientID     string
	ClientSecret string
	// PublicURL is where the browser reaches the gateway, such as
	// https://ivfs.example; the provider sends the browser back under it.
	PublicURL string
	// Group, when set, is the group a user must be in to sign in: the
	// provider decides who has an account, this decides who may use ivfs.
	Group string
	// LogoutURL, when set, is the provider's own sign-out page, for a
	// provider whose discovery document has no end_session_endpoint, such as
	// Authelia 4.39: without one, sign-out ends the gateway's session alone.
	LogoutURL string
	// HTTPClient is how the gateway reaches the provider; nil means
	// http.DefaultClient, which has no timeout and trusts only the system's
	// certificate authorities.
	HTTPClient *http.Client

	mu         sync.Mutex
	discovered *discovered
}

// discovered is what the gateway learns from the provider's discovery
// document.
type discovered struct {
	oauth2     oauth2.Config
	verifier   *oidc.IDTokenVerifier
	endSession string
}

// discover fetches the provider's discovery document the first time a user
// signs in, rather than at startup, so the gateway starts while the provider
// is down. A failure is not kept, so the next sign-in tries again.
func (p *Provider) discover(ctx context.Context) (*discovered, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.discovered != nil {
		return p.discovered, nil
	}

	// The key set keeps this context's client for every later key fetch.
	provider, err := oidc.NewProvider(p.clientContext(ctx), p.Issuer)
	if err != nil {
		return nil, fmt.Errorf("discover identity provider %s: %w", p.Issuer, err)
	}
	var metadata struct {
		EndSessionEndpoint string `json:"end_session_endpoint"`
	}
	if err := provider.Claims(&metadata); err != nil {
		return nil, fmt.Errorf("read discovery document of %s: %w", p.Issuer, err)
	}

	scopes := []string{oidc.ScopeOpenID}
	if p.Group != "" {
		scopes = append(scopes, "groups")
	}
	p.discovered = &discovered{
		oauth2: oauth2.Config{
			ClientID:     p.ClientID,
			ClientSecret: p.ClientSecret,
			Endpoint:     provider.Endpoint(),
			RedirectURL:  p.PublicURL + CallbackPath,
			Scopes:       scopes,
		},
		verifier:   provider.Verifier(&oidc.Config{ClientID: p.ClientID}),
		endSession: metadata.EndSessionEndpoint,
	}
	return p.discovered, nil
}

// clientContext returns ctx carrying HTTPClient, which go-oidc and oauth2
// both take from the context rather than from a parameter.
func (p *Provider) clientContext(ctx context.Context) context.Context {
	if p.HTTPClient == nil {
		return ctx
	}
	return oidc.ClientContext(ctx, p.HTTPClient)
}
