package probe

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestHandler checks that liveness ignores failing checks, and that
// readiness fails on any of them, naming each.
func TestHandler(t *testing.T) {
	healthy := func(context.Context) error { return nil }
	down := func(context.Context) error { return errors.New("connection refused") }

	tests := map[string]struct {
		checks     map[string]Check
		path       string
		wantStatus int
		wantBody   string
	}{
		"live while a dependency is down": {
			checks: map[string]Check{"registry": down}, path: "/livez",
			wantStatus: http.StatusOK, wantBody: "ok\n",
		},
		"ready when every check passes": {
			checks: map[string]Check{"registry": healthy, "temporal": healthy}, path: "/readyz",
			wantStatus: http.StatusOK, wantBody: "ok\n",
		},
		"not ready, naming every failure": {
			checks: map[string]Check{"temporal": down, "registry": down}, path: "/readyz",
			wantStatus: http.StatusServiceUnavailable,
			wantBody:   "registry: connection refused\ntemporal: connection refused\n",
		},
		"anything else": {
			checks: map[string]Check{}, path: "/",
			wantStatus: http.StatusNotFound, wantBody: "404 page not found\n",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			Handler(tt.checks).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantBody, rec.Body.String())
		})
	}
}
