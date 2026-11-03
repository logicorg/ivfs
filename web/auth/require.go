package auth

import (
	"net/http"
	"strings"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Require lets a request through only with a valid session, acting as the
// user it names: this is the one place a user enters a context (ADR 0013).
// Signing in, and the health check a load balancer polls, need no session.
//
// Without a session, an API request gets a 401, which the UI answers by
// sending the browser to sign in; a page load is sent there directly.
func Require(signer *Signer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/auth/") || r.URL.Path == "/api/health" {
				next.ServeHTTP(w, r)
				return
			}

			uid, err := signer.SessionUser(r, time.Now())
			if err != nil {
				if strings.HasPrefix(r.URL.Path, "/api/") {
					gateway.WriteProblem(w, http.StatusUnauthorized, err.Error()+"; sign in at "+LoginPath)
					return
				}
				http.Redirect(w, r, LoginPath, http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r.WithContext(tenancy.WithUser(r.Context(), uid)))
		})
	}
}
