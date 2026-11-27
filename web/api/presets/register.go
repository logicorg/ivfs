// Package presets is the web API's preset endpoints: the default preset, and
// applying one.
package presets

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"presets"}

// Register adds every preset endpoint to api.
func Register(api huma.API, c client.Client) {
	registerDefault(api)
	registerApply(api, c)
}
