// Package positions is the web API's position movement endpoints: for now, the
// deletion of an entry-less movement group (ADR 0011).
package positions

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"positions"}

// Register adds every position movement endpoint to api.
func Register(api huma.API, c client.Client) {
	registerDeleteMovement(api, c)
}
