// Package counterparties holds the ledger workflows that manage counterparties.
package counterparties

import (
	counterpartyactivity "github.com/logicorg/ivfs/finance/activities/ledger/counterparties"
)

// ca is a (nil) pointer to the counterparty activities struct. It is used to call the activities in the workflow.
var ca = (*counterpartyactivity.Activities)(nil)
