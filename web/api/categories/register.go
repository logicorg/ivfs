// Package categories is the web API's category endpoints, over the
// categories facade.
package categories

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"categories"}

// Register adds every category endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerCreate(api, c)
	registerRename(api, c)
	registerMove(api, c)
	registerClose(api, c)
	registerDelete(api, c)
	registerDeletionPreview(api, c)
	registerTotals(api, c)
}
