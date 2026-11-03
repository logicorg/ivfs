// Package currencies is the web API's currency endpoints, over the currency
// workflows.
package currencies

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"go.temporal.io/sdk/client"
)

var tags = []string{"currencies"}

// Register adds every currency endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerCreate(api, c)
	registerUpdate(api, c)
	registerDelete(api, c)
}

// RegisterRates adds the endpoint of a pair's conversion rates to api, over
// sources besides the workflows.
func RegisterRates(api huma.API, c client.Client, sources assetvalue.Sources) {
	registerRates(api, c, sources)
}
