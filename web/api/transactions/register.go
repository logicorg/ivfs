// Package transactions is the web API's transaction endpoints: an expense, an
// income, or a transfer, each recorded as one journal entry (ADR 0008).
package transactions

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"transactions"}

// Register adds every transaction endpoint to api.
func Register(api huma.API, c client.Client) {
	registerExpense(api, c)
	registerIncome(api, c)
	registerTransfer(api, c)
}
