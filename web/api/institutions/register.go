// Package institutions is the web API's institution endpoints.
package institutions

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"institutions"}

// Register adds every institution endpoint to api.
func Register(api huma.API, c client.Client) {
	registerClose(api, c)
	registerDelete(api, c)
	registerPreviewDeletion(api, c)
}
