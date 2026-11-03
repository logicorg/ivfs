package gateway

import (
	"mime"
	"net/http"
	"net/url"
	"strings"
)

// Guard keeps another site's page, open in the same browser, from using the
// API with the user's session (ADR 0007, ADR 0013). SameSite=Lax already
// keeps the session cookie off a cross-site fetch; this does not rely on it
// alone. It rejects:
//
//   - any request whose Host is not one the gateway answers to, which stops
//     DNS rebinding;
//   - an API request, or any request that is not a GET or HEAD, that the
//     browser marks as coming from another site, by Sec-Fetch-Site or by
//     Origin. That covers signing out, which a cross-site form could
//     otherwise do to the user;
//   - an API request with a body that is not JSON, which a cross-site form
//     can never send without a CORS preflight the gateway does not answer.
//
// A GET outside /api is not checked for its origin, so a link from another
// site can still open the UI, and the identity provider can send the
// browser back to /auth/callback.
func Guard(allowedHosts []string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(allowedHosts))
	for _, host := range allowedHosts {
		allowed[strings.ToLower(host)] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !allowed[strings.ToLower(r.Host)] {
				// Names the setting, not the allowed host, which is no business
				// of whoever sent the wrong one.
				WriteProblem(w, http.StatusForbidden, "unexpected host "+r.Host+
					"; the gateway answers only to the host of its IVFS_PUBLIC_URL")
				return
			}
			api := strings.HasPrefix(r.URL.Path, "/api/")
			if !api && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				next.ServeHTTP(w, r)
				return
			}

			switch r.Header.Get("Sec-Fetch-Site") {
			case "cross-site", "same-site":
				WriteProblem(w, http.StatusForbidden, "requests from another site are not allowed")
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				originURL, err := url.Parse(origin)
				if err != nil || !allowed[strings.ToLower(originURL.Host)] {
					WriteProblem(w, http.StatusForbidden, "requests from another site are not allowed")
					return
				}
			}
			if api && r.ContentLength != 0 && !isJSON(r.Header.Get("Content-Type")) {
				WriteProblem(w, http.StatusUnsupportedMediaType, "request bodies must be application/json")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func isJSON(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	return err == nil && mediaType == "application/json"
}
