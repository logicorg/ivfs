// Package gateway holds what every area of the web API shares (ADR 0007):
// running a workflow and turning its error into a problem, the guard against
// requests from other sites, and the shape of the OpenAPI document.
package gateway

import "github.com/danielgtaylor/huma/v2"

// Version is the API's version in its OpenAPI document.
const Version = "0.1.0"

// Arrays in the API are never null, so TypeScript sees Currency[] rather
// than Currency[] | null. An endpoint therefore returns an empty slice, never
// a nil one. Every area imports this package, so this runs before any of them
// registers an endpoint.
func init() {
	huma.DefaultArrayNullable = false
}

// NewConfig returns the Huma configuration of the API.
func NewConfig() huma.Config {
	config := huma.DefaultConfig("ivfs", Version)
	// The default create hook adds a $schema field to every response body,
	// which would become a field of every generated TypeScript type.
	config.CreateHooks = nil
	config.OpenAPIPath = "/api/openapi"
	config.DocsPath = "/api/docs"
	config.SchemasPath = "/api/schemas"
	return config
}
