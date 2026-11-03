package gateway

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
)

// NegativeHolding warns that an asset is held in negative units in a financial
// account, from the first point in time where that is so.
type NegativeHolding struct {
	FinancialAccountID   int64     `json:"financialAccountId"`
	FinancialAccountName string    `json:"financialAccountName"`
	AssetID              int64     `json:"assetId"`
	AssetName            string    `json:"assetName"`
	At                   time.Time `json:"at" doc:"when the holding first goes negative"`
	Units                Amount    `json:"units" doc:"the units held then"`
}

// NewNegativeHoldings is warnings as the API returns them.
func NewNegativeHoldings(warnings []lt.NegativeHolding) []NegativeHolding {
	if len(warnings) == 0 {
		return nil
	}
	out := make([]NegativeHolding, len(warnings))
	for n, w := range warnings {
		out[n] = NegativeHolding{
			FinancialAccountID: w.FinancialAccountID, FinancialAccountName: w.FinancialAccountName,
			AssetID: w.AssetID, AssetName: w.AssetName, At: w.At, Units: NewAmount(w.Units),
		}
	}
	return out
}

// Deletion is what a deletion leaves behind worth knowing, whether of a journal
// entry or of a position movement group.
type Deletion struct {
	// Warnings are the positions the deletion leaves negative at some point in
	// time. The deletion is made all the same (ADR 0011).
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

// AccountDeletion is what deleting an account or an institution answers.
type AccountDeletion struct {
	// WorkflowID is the deletion still running, answered with 202 Accepted.
	WorkflowID string `json:"workflowId,omitempty" doc:"the deletion still running, with 202 Accepted; deleting again waits on it"`
	// Warnings are the positions the deletion leaves negative at some point in
	// time.
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

// AccountDeletionOutput is the response of deleting an account or an
// institution: 200 once it is done, 202 while it is still running.
type AccountDeletionOutput struct {
	Status int
	Body   AccountDeletion
}
