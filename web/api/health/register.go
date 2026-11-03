// Package health is the web API's health endpoint.
package health

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

// Register adds every health endpoint to api.
func Register(api huma.API, c client.Client) {
	registerGet(api, c)
}
