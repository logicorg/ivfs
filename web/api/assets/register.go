// Package assets is the web API's asset endpoints, over the asset workflows:
// the instruments a brokerage account buys and sells.
package assets

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"go.temporal.io/sdk/client"
)

var tags = []string{"assets"}

// Register adds every asset endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerCreate(api, c)
}

// RegisterValues adds the endpoint of the asset's values to api, over
// sources besides the workflows.
func RegisterValues(api huma.API, c client.Client, sources assetvalue.Sources) {
	registerValues(api, c, sources)
}
