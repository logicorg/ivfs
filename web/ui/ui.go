// Package ui serves the web UI's production build, which the gateway embeds
// (ADR 0007). The build is the Vite project in this directory, written to
// dist/app by `make ui`.
package ui

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

// dist holds the build, and a committed placeholder so it embeds before the
// UI has been built.
//
//go:embed all:dist
var dist embed.FS

// Handler serves the embedded build.
func Handler() http.Handler {
	return handler(dist)
}

// handler serves the build in fsys's dist/app. A path that names a file is
// that file; any other path is index.html, so the UI's own router resolves it.
// Vite names each asset after its content, so assets are cached for good,
// while index.html is checked on every load to pick up a new build.
func handler(fsys fs.FS) http.Handler {
	app, err := fs.Sub(fsys, "dist/app")
	if err == nil {
		_, err = fs.Stat(app, "index.html")
	}
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "The UI has not been built into this binary: run make ui, then rebuild the gateway.",
				http.StatusServiceUnavailable)
		})
	}

	files := http.FileServerFS(app)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			files.ServeHTTP(w, r)
			return
		}
		if name != "" && name != "index.html" {
			if info, err := fs.Stat(app, name); err == nil && !info.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, app, "index.html")
	})
}
