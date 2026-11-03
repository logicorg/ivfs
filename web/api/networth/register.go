// Package networth is the web API's net worth endpoints (ADR 0015).
package networth

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"net worth"}

// Register adds every net worth endpoint to api.
func Register(api huma.API, c client.Client) {
	registerSeries(api, c)
}
