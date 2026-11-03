package main

import (
	"encoding/pem"
	"maps"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Two keys, as during a rotation.
	const keys = "a2V5LW9uZS1rZXktb25lLWtleS1vbmUta2V5LW9uZSE=, a2V5LXR3by1rZXktdHdvLWtleS10d28ta2V5LXR3byE="
	valid := map[string]string{
		publicURLVariable:    "https://ivfs.example",
		sessionKeysVariable:  keys,
		oidcIssuerVariable:   "https://auth.example",
		oidcClientIDVariable: "ivfs",
		oidcSecretVariable:   "secret",
	}

	tests := map[string]struct {
		env            map[string]string
		wantURL        string
		wantHost       string
		wantGroup      string
		wantLogoutURL  string
		wantProviders  []string
		wantTimeout    time.Duration
		wantErr        string
		wantErrMissing []string
	}{
		"the required variables": {
			wantURL: "https://ivfs.example", wantHost: "ivfs.example",
			wantProviders: []string{"manual"}, wantTimeout: defaultValueTimeout,
		},
		"a group": {
			env:     map[string]string{oidcGroupVariable: "ivfs-users"},
			wantURL: "https://ivfs.example", wantHost: "ivfs.example", wantGroup: "ivfs-users",
		},
		"a sign-out page": {
			env:     map[string]string{oidcLogoutVariable: "https://auth.example/logout"},
			wantURL: "https://ivfs.example", wantHost: "ivfs.example", wantLogoutURL: "https://auth.example/logout",
		},
		"a trailing slash": {
			env:     map[string]string{publicURLVariable: "https://ivfs.example/"},
			wantURL: "https://ivfs.example", wantHost: "ivfs.example",
		},
		"a port": {
			env:     map[string]string{publicURLVariable: "https://ivfs.example:8443"},
			wantURL: "https://ivfs.example:8443", wantHost: "ivfs.example:8443",
		},
		"the default port, which the browser leaves out of Host": {
			env:     map[string]string{publicURLVariable: "https://ivfs.example:443"},
			wantURL: "https://ivfs.example", wantHost: "ivfs.example",
		},
		"http on localhost": {
			env:     map[string]string{publicURLVariable: "http://localhost:8080"},
			wantURL: "http://localhost:8080", wantHost: "localhost:8080",
		},
		"http on IPv6 loopback": {
			env:     map[string]string{publicURLVariable: "http://[::1]"},
			wantURL: "http://[::1]", wantHost: "[::1]",
		},
		"nothing set": {
			env: map[string]string{
				publicURLVariable: "", sessionKeysVariable: "", oidcIssuerVariable: "",
				oidcClientIDVariable: "", oidcSecretVariable: "",
			},
			wantErrMissing: []string{publicURLVariable, oidcIssuerVariable, oidcClientIDVariable,
				oidcSecretVariable, sessionKeysVariable},
		},
		"no client secret": {
			env:            map[string]string{oidcSecretVariable: ""},
			wantErrMissing: []string{oidcSecretVariable},
		},
		"http on another host": {
			env:     map[string]string{publicURLVariable: "http://ivfs.example"},
			wantErr: "must use https",
		},
		"a path": {
			env:     map[string]string{publicURLVariable: "https://ivfs.example/ivfs"},
			wantErr: "must be an origin alone",
		},
		"a query": {
			env:     map[string]string{publicURLVariable: "https://ivfs.example?x=1"},
			wantErr: "must be an origin alone",
		},
		"a user in the URL": {
			env:     map[string]string{publicURLVariable: "https://someone@ivfs.example"},
			wantErr: "must be an origin alone",
		},
		"a host alone": {
			env:     map[string]string{publicURLVariable: "ivfs.example"},
			wantErr: "is not an absolute URL",
		},
		"a key that is not base64": {
			env:     map[string]string{sessionKeysVariable: "not base64!"},
			wantErr: "key 0 is not base64",
		},
		"a short key": {
			env:     map[string]string{sessionKeysVariable: keys + ",c2hvcnQ="},
			wantErr: "signing key 2 is 5 bytes long",
		},
		"an empty key in the list": {
			env:     map[string]string{sessionKeysVariable: keys + ","},
			wantErr: "signing key 2 is 0 bytes long",
		},
		"a relative sign-out page": {
			env:     map[string]string{oidcLogoutVariable: "/logout"},
			wantErr: "invalid OIDC_LOGOUT_URL: \"/logout\" is not an absolute URL",
		},
		"a sign-out page that is not http": {
			env:     map[string]string{oidcLogoutVariable: "javascript:alert(1)"},
			wantErr: "invalid OIDC_LOGOUT_URL",
		},
		"a CA file that does not exist": {
			env:     map[string]string{oidcCAFileVariable: "/nonexistent/ca.pem"},
			wantErr: "invalid OIDC_CA_FILE: open /nonexistent/ca.pem",
		},
		"value providers": {
			env: map[string]string{
				valueProvidersVariable: "yahoo=http://yahoo-provider:8080,manual",
				valueTimeoutVariable:   "2s",
			},
			wantURL: "https://ivfs.example", wantHost: "ivfs.example",
			wantProviders: []string{"yahoo", "manual"}, wantTimeout: 2 * time.Second,
		},
		"a value provider without a URL": {
			env:     map[string]string{valueProvidersVariable: "yahoo"},
			wantErr: "invalid IVFS_ASSET_VALUE_PROVIDERS: provider yahoo has no URL",
		},
		"a value provider timeout that is not a duration": {
			env:     map[string]string{valueTimeoutVariable: "5"},
			wantErr: "invalid IVFS_ASSET_VALUE_PROVIDER_TIMEOUT: \"5\" is not a duration, such as 5s",
		},
		"a value provider timeout of zero": {
			env:     map[string]string{valueTimeoutVariable: "0s"},
			wantErr: "invalid IVFS_ASSET_VALUE_PROVIDER_TIMEOUT: 0s must be positive, or unset for 5s",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			env := maps.Clone(valid)
			maps.Copy(env, tc.env)

			cfg, err := loadConfig(func(name string) string { return env[name] })
			switch {
			case tc.wantErrMissing != nil:
				require.EqualError(t, err, "set the environment variables "+strings.Join(tc.wantErrMissing, ", "))
				return
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantURL, cfg.publicURL)
			require.Equal(t, tc.wantHost, cfg.publicHost)
			require.Equal(t, tc.wantGroup, cfg.group)
			require.Equal(t, tc.wantLogoutURL, cfg.logoutURL)
			require.NotNil(t, cfg.signer)
			require.NotNil(t, cfg.providerClient)
			if tc.wantProviders != nil {
				require.Equal(t, tc.wantProviders, providerNames(cfg.valueProviders))
				require.Equal(t, tc.wantTimeout, cfg.valueTimeout)
			}
		})
	}
}

// TestProviderClient reaches a TLS server whose certificate no system
// authority signed, as the development compose's provider is.
func TestProviderClient(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(server.Close)
	dir := t.TempDir()
	write := func(name string, content []byte) string {
		path := filepath.Join(dir, name)
		require.NoError(t, os.WriteFile(path, content, 0o600))
		return path
	}
	serverCA := write("server.pem", pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw}))
	notPEM := write("not.pem", []byte("not a certificate"))

	tests := map[string]struct {
		caFile         string
		wantErr        string
		wantRequestErr string
	}{
		"the server's authority": {caFile: serverCA},
		"no CA file, so only the system's authorities": {
			wantRequestErr: "certificate",
		},
		"a CA file that does not exist": {
			caFile:  filepath.Join(dir, "missing.pem"),
			wantErr: "no such file",
		},
		"a CA file without a certificate": {
			caFile:  notPEM,
			wantErr: notPEM + " holds no PEM certificate",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := providerClient(tc.caFile)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, providerTimeout, client.Timeout)

			resp, err := client.Get(server.URL)
			if tc.wantRequestErr != "" {
				require.ErrorContains(t, err, tc.wantRequestErr)
				return
			}
			require.NoError(t, err)
			require.NoError(t, resp.Body.Close())
		})
	}
}

func TestParseValueProviders(t *testing.T) {
	tests := map[string]struct {
		raw         string
		wantNames   []string
		wantClients []bool
		wantErr     string
	}{
		"unset, so manual alone": {
			wantNames: []string{"manual"}, wantClients: []bool{false},
		},
		"blank, so manual alone": {
			raw: "  ", wantNames: []string{"manual"}, wantClients: []bool{false},
		},
		"manual and an external one, in order": {
			raw:       "manual,yahoo=http://yahoo-provider:8080",
			wantNames: []string{"manual", "yahoo"}, wantClients: []bool{false, true},
		},
		"external ones alone, with spaces": {
			raw:       " yahoo = http://yahoo-provider:8080 , other-2=https://other.example/prices ",
			wantNames: []string{"yahoo", "other-2"}, wantClients: []bool{true, true},
		},
		"an empty entry": {
			raw:     "manual,,yahoo=http://yahoo-provider:8080",
			wantErr: "provider 1 has no name; list them as manual,yahoo=http://yahoo-provider:8080",
		},
		"a URL without a name": {
			raw:     "=http://yahoo-provider:8080",
			wantErr: "provider 0 has no name",
		},
		"a name with capitals": {
			raw:     "Yahoo=http://yahoo-provider:8080",
			wantErr: `provider name "Yahoo" must be lowercase letters, digits and dashes`,
		},
		"a name listed twice": {
			raw:     "yahoo=http://a:8080,yahoo=http://b:8080",
			wantErr: "provider yahoo is listed twice",
		},
		"manual listed twice": {
			raw:     "manual,manual",
			wantErr: "provider manual is listed twice",
		},
		"manual with a URL": {
			raw:     "manual=http://manual:8080",
			wantErr: "provider manual is built in and takes no URL; list it as manual alone",
		},
		"an external one without a URL": {
			raw:     "yahoo",
			wantErr: "provider yahoo has no URL; list it as yahoo=http://yahoo-provider:8080",
		},
		"a relative URL": {
			raw:     "yahoo=yahoo-provider:8080",
			wantErr: "provider yahoo: \"yahoo-provider:8080\" is not an absolute URL",
		},
		"a URL with a query": {
			raw:     "yahoo=http://yahoo-provider:8080?key=1",
			wantErr: "provider yahoo: \"http://yahoo-provider:8080?key=1\" must be a base URL alone",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			providers, err := parseValueProviders(tc.raw)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantNames, providerNames(providers))
			clients := make([]bool, len(providers))
			for i, provider := range providers {
				clients[i] = provider.client != nil
			}
			require.Equal(t, tc.wantClients, clients)
		})
	}
}

func providerNames(providers []valueProvider) []string {
	names := make([]string, len(providers))
	for i, provider := range providers {
		names[i] = provider.name
	}
	return names
}
