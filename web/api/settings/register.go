// Package settings is the web API's endpoints over the user's settings
// (ADR 0015).
package settings

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"settings"}

// Register adds every settings endpoint to api.
func Register(api huma.API, c client.Client) {
	registerRetrieve(api, c)
	registerUpdate(api, c)
}
