// Package tags is the web API's tag endpoints.
package tags

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"tags"}

// Register adds every tag endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerCreate(api, c)
	registerUpdate(api, c)
	registerDelete(api, c)
}
