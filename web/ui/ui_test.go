package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

// TestHandler checks that files and assets are served as they are, that any
// other path gets index.html for the UI's router, with the caching each
// needs, and that a binary without a build says so.
func TestHandler(t *testing.T) {
	built := fstest.MapFS{
		"dist/.gitkeep":                 {},
		"dist/app/index.html":           {Data: []byte("<html>index</html>")},
		"dist/app/favicon.svg":          {Data: []byte("<svg/>")},
		"dist/app/assets/index-ab12.js": {Data: []byte("console.log(1)")},
	}
	notBuilt := fstest.MapFS{"dist/.gitkeep": {}}

	tests := map[string]struct {
		fsys        fstest.MapFS
		path        string
		wantStatus  int
		wantBody    string
		wantCaching string
	}{
		"the root": {
			fsys: built, path: "/",
			wantStatus: http.StatusOK, wantBody: "<html>index</html>", wantCaching: "no-cache",
		},
		"a route of the UI": {
			fsys: built, path: "/accounts/3",
			wantStatus: http.StatusOK, wantBody: "<html>index</html>", wantCaching: "no-cache",
		},
		"index.html by name, which redirects to the root": {
			fsys: built, path: "/index.html",
			wantStatus: http.StatusMovedPermanently,
		},
		"a file": {
			fsys: built, path: "/favicon.svg",
			wantStatus: http.StatusOK, wantBody: "<svg/>",
		},
		"an asset": {
			fsys: built, path: "/assets/index-ab12.js",
			wantStatus: http.StatusOK, wantBody: "console.log(1)", wantCaching: "public, max-age=31536000, immutable",
		},
		"an asset that does not exist": {
			fsys: built, path: "/assets/index-old.js",
			wantStatus: http.StatusNotFound,
		},
		"a binary without a build": {
			fsys: notBuilt, path: "/",
			wantStatus: http.StatusServiceUnavailable, wantBody: "The UI has not been built into this binary: run make ui, then rebuild the gateway.\n",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler(tc.fsys).ServeHTTP(response, httptest.NewRequest(http.MethodGet, tc.path, nil))

			require.Equal(t, tc.wantStatus, response.Code)
			if tc.wantBody != "" {
				require.Equal(t, tc.wantBody, response.Body.String())
			}
			if tc.wantCaching != "" {
				require.Equal(t, tc.wantCaching, response.Header().Get("Cache-Control"))
			}
		})
	}
}
