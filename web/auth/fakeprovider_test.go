package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
)

const (
	clientID  = "ivfs"
	publicURL = "https://ivfs.example"
)

// fakeProvider is an OpenID Connect provider that issues an ID token for
// whatever grant a test authorizes, checking the PKCE verifier as a real
// one would. It serves TLS under its own certificate authority, so a call
// to it that skips Provider.HTTPClient fails.
type fakeProvider struct {
	server *httptest.Server
	key    *rsa.PrivateKey
	// endSession is the end_session_endpoint the discovery document
	// lists, if any.
	endSession string
	// down makes every endpoint answer 503.
	down atomic.Bool

	mu     sync.Mutex
	grants map[string]grant
}

// A grant is what the user approved at the provider, and the ID token the
// code is redeemed for.
type grant struct {
	challenge string
	nonce     string
	subject   string
	groups    []string
	audience  string
	// key signs the ID token; nil means the provider's own key.
	key *rsa.PrivateKey
}

func newFakeProvider(t *testing.T, endSession bool) *fakeProvider {
	t.Helper()
	idp := &fakeProvider{key: newKey(t), grants: map[string]grant{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", idp.discovery)
	mux.HandleFunc("GET /jwks", idp.jwks)
	mux.HandleFunc("POST /token", idp.token)
	idp.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if idp.down.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(idp.server.Close)
	if endSession {
		idp.endSession = idp.server.URL + "/logout?keep=1"
	}
	return idp
}

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

func (f *fakeProvider) issuer() string { return f.server.URL }

func (f *fakeProvider) discovery(w http.ResponseWriter, _ *http.Request) {
	doc := map[string]any{
		"issuer":                                f.issuer(),
		"authorization_endpoint":                f.issuer() + "/authorize",
		"token_endpoint":                        f.issuer() + "/token",
		"jwks_uri":                              f.issuer() + "/jwks",
		"id_token_signing_alg_values_supported": []string{"RS256"},
	}
	if f.endSession != "" {
		doc["end_session_endpoint"] = f.endSession
	}
	_ = json.NewEncoder(w).Encode(doc)
}

func (f *fakeProvider) jwks(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
		{Key: &f.key.PublicKey, KeyID: "test", Algorithm: "RS256", Use: "sig"},
	}})
}

func (f *fakeProvider) token(w http.ResponseWriter, r *http.Request) {
	refuse := func() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
	}
	if err := r.ParseForm(); err != nil {
		refuse()
		return
	}
	f.mu.Lock()
	g, ok := f.grants[r.PostForm.Get("code")]
	delete(f.grants, r.PostForm.Get("code"))
	f.mu.Unlock()
	if !ok || s256(r.PostForm.Get("code_verifier")) != g.challenge {
		refuse()
		return
	}

	key := g.key
	if key == nil {
		key = f.key
	}
	audience := g.audience
	if audience == "" {
		audience = clientID
	}
	claims := map[string]any{
		"iss": f.issuer(), "sub": g.subject, "aud": audience, "nonce": g.nonce,
		"iat": time.Now().Unix(), "exp": time.Now().Add(time.Minute).Unix(),
	}
	if g.groups != nil {
		claims["groups"] = g.groups
	}
	payload, _ := json.Marshal(claims)
	signer, _ := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key},
		(&jose.SignerOptions{}).WithHeader("kid", "test"))
	signed, _ := signer.Sign(payload)
	idToken, _ := signed.CompactSerialize()

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"access_token": "unused", "token_type": "Bearer", "expires_in": 60, "id_token": idToken,
	})
}

// authorize approves the sign-in authURL asks for, as the user would at the
// provider, after edit changes the grant, and returns the code and state the
// provider sends back.
func (f *fakeProvider) authorize(t *testing.T, authURL string, edit func(*grant)) (code, state string) {
	t.Helper()
	u, err := url.Parse(authURL)
	require.NoError(t, err)
	q := u.Query()
	require.Equal(t, "S256", q.Get("code_challenge_method"))

	g := grant{challenge: q.Get("code_challenge"), nonce: q.Get("nonce"), subject: "alice"}
	if edit != nil {
		edit(&g)
	}
	code = rand.Text()
	f.mu.Lock()
	f.grants[code] = g
	f.mu.Unlock()
	return code, q.Get("state")
}

func s256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
