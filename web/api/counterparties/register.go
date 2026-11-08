// Package counterparties is the web API's counterparty endpoints.
package counterparties

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"counterparties"}

// Register adds every counterparty endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerCreate(api, c)
	registerRename(api, c)
	registerDelete(api, c)
}
