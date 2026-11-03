package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/logicorg/ivfs/web/auth"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/mocks"
)

// TestNewHandler checks that a path outside /api reaches the UI, that an
// API path with no endpoint, or with no endpoint for the method, is a problem
// rather than the UI's page, and that without a session only signing in and
// the health check are served.
func TestNewHandler(t *testing.T) {
	wantBody := "the UI"

	app := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(wantBody))
	})

	tests := map[string]struct {
		method       string
		path         string
		signedOut    bool
		wantStatus   int
		wantAllow    []string
		wantLocation string
		wantBody     string
	}{
		"the UI's root": {
			method:     http.MethodGet,
			path:       "/",
			wantStatus: http.StatusOK,
			wantBody:   wantBody,
		},
		"a route of the UI": {
			method:     http.MethodGet,
			path:       "/accounts/3",
			wantStatus: http.StatusOK,
			wantBody:   wantBody,
		},
		"an API path with no endpoint": {
			method:     http.MethodGet,
			path:       "/api/nothing",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"no endpoint /api/nothing"}` + "\n",
		},
		"an API path with no endpoint for the method": {
			method:     http.MethodDelete,
			path:       "/api/currencies",
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  []string{http.MethodGet, http.MethodPost},
			wantBody: `{"title":"Method Not Allowed","status":405,` +
				`"detail":"DELETE is not allowed on /api/currencies"}` + "\n",
		},
		"a path under /auth with no endpoint": {
			method:     http.MethodGet,
			path:       "/auth/nothing",
			signedOut:  true,
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"no endpoint /auth/nothing"}` + "\n",
		},
		"signing out by link": {
			method:     http.MethodGet,
			path:       auth.LogoutPath,
			wantStatus: http.StatusMethodNotAllowed,
			wantAllow:  []string{http.MethodPost},
		},
		"the UI without a session": {
			method:       http.MethodGet,
			path:         "/accounts/3",
			signedOut:    true,
			wantStatus:   http.StatusSeeOther,
			wantLocation: auth.LoginPath,
		},
		"the API without a session": {
			method:     http.MethodGet,
			path:       "/api/currencies",
			signedOut:  true,
			wantStatus: http.StatusUnauthorized,
		},
		"an API path with no endpoint, without a session": {
			method:     http.MethodGet,
			path:       "/api/nothing",
			signedOut:  true,
			wantStatus: http.StatusUnauthorized,
		},
		// The provider is unreachable, so signing in reaches its endpoint
		// and fails there, rather than being sent to sign in.
		"signing in without a session": {
			method:     http.MethodGet,
			path:       auth.LoginPath,
			signedOut:  true,
			wantStatus: http.StatusBadGateway,
		},
	}

	signer := apitest.NewSigner(t)
	idp := &auth.Provider{Issuer: "http://127.0.0.1:1", ClientID: "ivfs", PublicURL: "http://" + apitest.Host}
	ensure := func(context.Context, string, string) (uuid.UUID, error) {
		t.Error("ensure ran")
		return uuid.Nil, nil
	}
	handler := NewHandler(&mocks.Client{}, []string{apitest.Host}, signer, auth.NewHandlers(signer, idp, ensure), app, assetvalue.Sources{})

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			request := httptest.NewRequest(tc.method, "http://"+apitest.Host+tc.path, nil)
			if !tc.signedOut {
				apitest.SignIn(request, signer)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)

			require.Equal(t, tc.wantStatus, response.Code, response.Body.String())
			require.Equal(t, tc.wantAllow, response.Header().Values("Allow"))
			require.Equal(t, tc.wantLocation, response.Header().Get("Location"))
			if tc.wantBody != "" {
				require.Equal(t, tc.wantBody, response.Body.String())
			}
		})
	}
}

// TestSpecIsUpToDate checks that the committed openapi.json is the document
// the API generates, so a change to an endpoint cannot reach the UI without
// regenerating it.
func TestSpecIsUpToDate(t *testing.T) {
	committed, err := os.ReadFile("openapi.json")
	require.NoError(t, err)

	generated, err := Spec()
	require.NoError(t, err)
	require.Equal(t, string(committed), string(generated), "openapi.json is out of date; run make api")
}
