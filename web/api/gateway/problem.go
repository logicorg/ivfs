package gateway

import (
	"encoding/json"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

// WriteProblem answers with an RFC 9457 problem, in the shape every API
// error has, for a request that never reaches an endpoint.
func WriteProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(huma.ErrorModel{
		Title:  http.StatusText(status),
		Status: status,
		Detail: detail,
	})
}
