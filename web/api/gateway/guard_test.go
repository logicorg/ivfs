package gateway

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// testHost is the Host every test request is sent to, and the only one the
// guard allows.
const testHost = "localhost:8080"

// TestGuard checks which requests reach the handler behind the guard, and
// that every rejected one is answered with a problem.
func TestGuard(t *testing.T) {
	tests := map[string]struct {
		method     string
		path       string
		host       string
		headers    map[string]string
		body       string
		wantStatus int
	}{
		"a same-origin API read": {
			path: "/api/health", headers: map[string]string{"Sec-Fetch-Site": "same-origin"},
			wantStatus: http.StatusOK,
		},
		"an API read from outside a browser": {
			path:       "/api/health",
			wantStatus: http.StatusOK,
		},
		"an API request typed into the address bar": {
			path: "/api/health", headers: map[string]string{"Sec-Fetch-Site": "none"},
			wantStatus: http.StatusOK,
		},
		"a JSON command from the UI": {
			method: http.MethodPost, path: "/api/currencies",
			headers: map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://localhost:8080", "Content-Type": "application/json; charset=utf-8"},
			body:    `{}`, wantStatus: http.StatusOK,
		},
		"a command with no body": {
			method: http.MethodPost, path: "/api/financial-accounts/1/close",
			wantStatus: http.StatusOK,
		},
		"another host, such as after DNS rebinding": {
			path: "/api/health", host: "attacker.example:8080",
			wantStatus: http.StatusForbidden,
		},
		"another host on a UI page": {
			path: "/", host: "attacker.example:8080",
			wantStatus: http.StatusForbidden,
		},
		"an API request from another site": {
			path: "/api/health", headers: map[string]string{"Sec-Fetch-Site": "cross-site"},
			wantStatus: http.StatusForbidden,
		},
		"an API request from another port of the same site": {
			path: "/api/health", headers: map[string]string{"Sec-Fetch-Site": "same-site"},
			wantStatus: http.StatusForbidden,
		},
		"an API request with another site's origin": {
			method: http.MethodPost, path: "/api/currencies",
			headers: map[string]string{"Origin": "https://attacker.example", "Content-Type": "application/json"},
			body:    `{}`, wantStatus: http.StatusForbidden,
		},
		"an API request with an opaque origin": {
			method: http.MethodPost, path: "/api/currencies",
			headers: map[string]string{"Origin": "null", "Content-Type": "application/json"},
			body:    `{}`, wantStatus: http.StatusForbidden,
		},
		"a form post": {
			method: http.MethodPost, path: "/api/currencies",
			headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			body:    "code=EUR", wantStatus: http.StatusUnsupportedMediaType,
		},
		"a text body": {
			method: http.MethodPost, path: "/api/currencies",
			headers: map[string]string{"Content-Type": "text/plain"},
			body:    `{}`, wantStatus: http.StatusUnsupportedMediaType,
		},
		"a UI page linked from another site": {
			path: "/", headers: map[string]string{"Sec-Fetch-Site": "cross-site"},
			wantStatus: http.StatusOK,
		},
		"the identity provider sending the browser back": {
			path: "/auth/callback?code=c&state=s", headers: map[string]string{"Sec-Fetch-Site": "cross-site"},
			wantStatus: http.StatusOK,
		},
		"signing out from the UI": {
			method: http.MethodPost, path: "/auth/logout",
			headers:    map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://localhost:8080"},
			wantStatus: http.StatusOK,
		},
		"signing out from a form on another site": {
			method: http.MethodPost, path: "/auth/logout",
			headers:    map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://attacker.example"},
			wantStatus: http.StatusForbidden,
		},
		"signing out from another site, in a browser without Sec-Fetch-Site": {
			method: http.MethodPost, path: "/auth/logout",
			headers:    map[string]string{"Origin": "https://attacker.example"},
			wantStatus: http.StatusForbidden,
		},
		"a form body outside the API": {
			method: http.MethodPost, path: "/auth/logout",
			headers: map[string]string{"Content-Type": "application/x-www-form-urlencoded"},
			body:    "x=1", wantStatus: http.StatusOK,
		},
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	handler := Guard([]string{testHost})(next)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			method, host := tc.method, tc.host
			if method == "" {
				method = http.MethodGet
			}
			if host == "" {
				host = testHost
			}
			request := httptest.NewRequest(method, "http://"+host+tc.path, strings.NewReader(tc.body))
			for key, value := range tc.headers {
				request.Header.Set(key, value)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(t, tc.wantStatus, response.Code)
			if tc.wantStatus != http.StatusOK {
				require.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
				require.Contains(t, response.Body.String(), `"status":`)
			}
		})
	}
}
