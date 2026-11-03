// Package probe serves the health endpoints Kubernetes probes: /livez, which
// answers while the process runs, and /readyz, which also checks what the
// process depends on. Liveness checks nothing on purpose: a database or
// Temporal outage is not fixed by restarting the process, only waited out.
package probe

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"
)

// Check reports whether one dependency is usable.
type Check func(ctx context.Context) error

// Handler serves /livez and /readyz, running every check on /readyz, by name.
func Handler(checks map[string]Check) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /livez", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		var failures []string
		for _, name := range slices.Sorted(maps.Keys(checks)) {
			if err := checks[name](r.Context()); err != nil {
				failures = append(failures, fmt.Sprintf("%s: %v", name, err))
			}
		}
		if len(failures) > 0 {
			http.Error(w, strings.Join(failures, "\n"), http.StatusServiceUnavailable)
			return
		}
		_, _ = fmt.Fprintln(w, "ok")
	})
	return mux
}
