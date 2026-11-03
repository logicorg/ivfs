// Package holdings is the web API's read side of investments: the units held
// of each asset in each financial account with what they cost, and the
// movements that add up to them.
package holdings

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"holdings"}

// Register adds every holding endpoint to api.
func Register(api huma.API, c client.Client) {
	registerList(api, c)
	registerListMovements(api, c)
}
