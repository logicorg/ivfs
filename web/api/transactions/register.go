// Package transactions is the web API's transaction endpoints: an expense, an
// income, or a transfer, each recorded as one journal entry or an entry
// corrected to one (ADR 0008).
package transactions

import (
	"github.com/danielgtaylor/huma/v2"
	"go.temporal.io/sdk/client"
)

var tags = []string{"transactions"}

// correctionDescription is what every correction does, whatever the entry was
// before.
const correctionDescription = "The posted entry becomes this transaction whatever it was before, and keeps its id and " +
	"its idempotency key; a draft is refused. It loses any position movement it had, and the holdings that leaves negative " +
	"at some point in time are returned as warnings; the correction is made all the same."

// Register adds every transaction endpoint to api.
func Register(api huma.API, c client.Client) {
	registerExpense(api, c)
	registerIncome(api, c)
	registerTransfer(api, c)
	registerUpdateExpense(api, c)
	registerUpdateIncome(api, c)
	registerUpdateTransfer(api, c)
}
