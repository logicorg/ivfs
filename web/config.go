package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/logicorg/ivfs/web/providers/manual"
)

// The environment the gateway is configured from (ADR 0013).
const (
	publicURLVariable    = "IVFS_PUBLIC_URL"
	sessionKeysVariable  = "IVFS_SESSION_KEYS"
	oidcIssuerVariable   = "OIDC_ISSUER"
	oidcClientIDVariable = "OIDC_CLIENT_ID"
	oidcSecretVariable   = "OIDC_CLIENT_SECRET" //nolint: gosec
	oidcGroupVariable    = "OIDC_GROUP"
	oidcCAFileVariable   = "OIDC_CA_FILE"
	oidcLogoutVariable   = "OIDC_LOGOUT_URL"
	// The asset value providers (ADR 0014).
	valueProvidersVariable = "IVFS_ASSET_VALUE_PROVIDERS"
	valueTimeoutVariable   = "IVFS_ASSET_VALUE_PROVIDER_TIMEOUT"
)

// providerTimeout bounds each call to the identity provider: a sign-in waits
// on it, and a stalled provider would otherwise hold the sign-in forever.
const providerTimeout = 10 * time.Second

// defaultValueTimeout bounds each asset value provider's answer, so a slow
// one delays a price by seconds, not the gateway's whole timeout.
const defaultValueTimeout = 5 * time.Second

// valueProvider is an asset value provider as configured: client is nil for
// the built-in manual one, which needs the Temporal client to be built.
type valueProvider struct {
	name   string
	client *assetvalue.Client
}

// config is what the gateway needs to sign users in.
type config struct {
	// publicURL is the gateway's origin as the browser sees it, with no
	// trailing slash, and publicHost the Host header it sends there.
	publicURL, publicHost string
	signer                *auth.Signer
	issuer, clientID      string
	clientSecret, group   string
	logoutURL             string
	// providerClient is how the gateway reaches the identity provider.
	providerClient *http.Client
	// valueProviders are in order of preference, and valueTimeout bounds
	// each one's answer.
	valueProviders []valueProvider
	valueTimeout   time.Duration
}

// loadConfig reads the configuration through getenv, reporting every
// missing variable at once rather than one per restart.
func loadConfig(getenv func(string) string) (config, error) {
	var missing []string
	required := func(name string) string {
		value := getenv(name)
		if value == "" {
			missing = append(missing, name)
		}
		return value
	}
	cfg := config{
		publicURL:    required(publicURLVariable),
		issuer:       required(oidcIssuerVariable),
		clientID:     required(oidcClientIDVariable),
		clientSecret: required(oidcSecretVariable),
		group:        getenv(oidcGroupVariable),
	}
	keys := required(sessionKeysVariable)
	if len(missing) > 0 {
		return config{}, fmt.Errorf("set the environment variables %s", strings.Join(missing, ", "))
	}

	var err error
	if cfg.publicURL, cfg.publicHost, err = parsePublicURL(cfg.publicURL); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", publicURLVariable, err)
	}
	if cfg.signer, err = parseKeys(keys); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", sessionKeysVariable, err)
	}
	if cfg.providerClient, err = providerClient(getenv(oidcCAFileVariable)); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", oidcCAFileVariable, err)
	}
	if cfg.logoutURL, err = parseLogoutURL(getenv(oidcLogoutVariable)); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", oidcLogoutVariable, err)
	}
	if cfg.valueProviders, err = parseValueProviders(getenv(valueProvidersVariable)); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", valueProvidersVariable, err)
	}
	if cfg.valueTimeout, err = parseValueTimeout(getenv(valueTimeoutVariable)); err != nil {
		return config{}, fmt.Errorf("invalid %s: %w", valueTimeoutVariable, err)
	}
	return cfg, nil
}

// providerName is what a provider can be called: the UI shows the name and
// remembers it per asset, so it is kept plain.
var providerName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// parseValueProviders reads a comma-separated list of providers, as
// manual,yahoo=http://yahoo-provider:8080. Unset, it is the manual provider
// alone, which is what ivfs offered before there were others.
func parseValueProviders(raw string) ([]valueProvider, error) {
	if strings.TrimSpace(raw) == "" {
		return []valueProvider{{name: manual.Name}}, nil
	}
	// The client has no timeout of its own: each call is bounded by its
	// context, with the configured timeout.
	httpClient := &http.Client{}
	var providers []valueProvider
	seen := make(map[string]bool)
	for i, entry := range strings.Split(raw, ",") {
		name, baseURL, hasURL := strings.Cut(strings.TrimSpace(entry), "=")
		name = strings.TrimSpace(name)
		switch {
		case name == "":
			return nil, fmt.Errorf("provider %d has no name; list them as manual,yahoo=http://yahoo-provider:8080", i)
		case !providerName.MatchString(name):
			return nil, fmt.Errorf("provider name %q must be lowercase letters, digits and dashes", name)
		case seen[name]:
			return nil, fmt.Errorf("provider %s is listed twice", name)
		case name == manual.Name && hasURL:
			return nil, fmt.Errorf("provider %s is built in and takes no URL; list it as %s alone", name, name)
		case name != manual.Name && !hasURL:
			return nil, fmt.Errorf("provider %s has no URL; list it as %s=http://%s-provider:8080", name, name, name)
		}
		seen[name] = true

		provider := valueProvider{name: name}
		if hasURL {
			client, err := assetvalue.NewClient(strings.TrimSpace(baseURL), httpClient)
			if err != nil {
				return nil, fmt.Errorf("provider %s: %w", name, err)
			}
			provider.client = client
		}
		providers = append(providers, provider)
	}
	return providers, nil
}

// parseValueTimeout reads a duration such as 5s, defaulting to
// defaultValueTimeout.
func parseValueTimeout(raw string) (time.Duration, error) {
	if raw == "" {
		return defaultValueTimeout, nil
	}
	timeout, err := time.ParseDuration(raw)
	switch {
	case err != nil:
		return 0, fmt.Errorf("%q is not a duration, such as 5s", raw)
	case timeout <= 0:
		return 0, fmt.Errorf("%s must be positive, or unset for %s", raw, defaultValueTimeout)
	}
	return timeout, nil
}

// parseLogoutURL checks that raw, when set, is where a browser can be sent:
// a relative one would land on the gateway, which signs the user straight
// back in.
func parseLogoutURL(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return "", err
	case (u.Scheme != "https" && u.Scheme != "http") || u.Host == "":
		return "", fmt.Errorf("%q is not an absolute URL, such as https://auth.example/logout", raw)
	}
	return raw, nil
}

// parsePublicURL returns raw as an origin, and the Host header a browser
// sends to it. The session cookie is Secure, so only HTTPS can carry it,
// except on loopback, which browsers treat as secure for local development.
func parsePublicURL(raw string) (origin, host string, err error) {
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return "", "", err
	case u.Host == "":
		return "", "", fmt.Errorf("%q is not an absolute URL, such as https://ivfs.example", raw)
	case strings.TrimSuffix(u.Path, "/") != "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil:
		return "", "", fmt.Errorf("%q must be an origin alone, such as https://ivfs.example: "+
			"the gateway serves from the root", raw)
	}

	hostname, port := u.Hostname(), u.Port()
	switch {
	case u.Scheme == "https":
		if port == "443" {
			port = ""
		}
	case u.Scheme == "http" && isLoopback(hostname):
		if port == "80" {
			port = ""
		}
	default:
		return "", "", fmt.Errorf("%q must use https, which the session cookie needs, "+
			"or http on localhost", raw)
	}

	// A browser leaves the scheme's default port out of the Host header.
	host = hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return u.Scheme + "://" + host, host, nil
}

func isLoopback(hostname string) bool {
	if hostname == "localhost" {
		return true
	}
	ip := net.ParseIP(hostname)
	return ip != nil && ip.IsLoopback()
}

// parseKeys returns a signer over comma-separated base64 keys, the newest
// first, as `openssl rand -base64 32` prints them.
func parseKeys(raw string) (*auth.Signer, error) {
	var keys [][]byte
	for i, encoded := range strings.Split(raw, ",") {
		key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(encoded))
		if err != nil {
			return nil, fmt.Errorf("key %d is not base64; generate one with `openssl rand -base64 %d`", i, auth.MinKeyLen)
		}
		keys = append(keys, key)
	}
	return auth.NewSigner(keys)
}

// providerClient returns the client the gateway reaches the identity
// provider with. caFile, when set, is a PEM file of certificate authorities
// to trust besides the system's, for a provider with a private one, such as
// the development compose's. Go ignores SSL_CERT_FILE on macOS, so this is
// the way that works everywhere.
func providerClient(caFile string) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if caFile != "" {
		pem, err := os.ReadFile(caFile)
		if err != nil {
			return nil, err
		}
		roots, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("load the system's certificate authorities: %w", err)
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("%s holds no PEM certificate", caFile)
		}
		transport.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{Transport: transport, Timeout: providerTimeout}, nil
}
