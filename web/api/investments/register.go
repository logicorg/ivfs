// Package investments is the web API's investment endpoints: a buy, a sale or
// a vest in a brokerage account, each recorded as one journal entry and one
// position movement (ADR 0009).
package investments

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"investments"}

// Register adds every investment endpoint to api.
func Register(api huma.API, c client.Client) {
	registerBuy(api, c)
	registerSell(api, c)
	registerVest(api, c)
}
