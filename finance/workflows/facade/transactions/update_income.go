package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"go.temporal.io/sdk/workflow"
)

// UpdateIncome makes a journal entry an income with the given fields,
// whatever it was before, with the checks of RecordIncome. The entry keeps its
// id, and loses any position movement it had, so the holdings left negative
// are returned as warnings. It fails with ErrInvalidParameter, writing
// nothing, for what RecordIncome refuses, and with ErrNotFound when the entry
// does not exist.
func UpdateIncome(ctx workflow.Context, input tt.UpdateIncomeInput) (*tt.UpdateResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	m, err := incomeMovement(ctx, input.Income())
	if err != nil {
		return nil, err
	}
	return replace(ctx, input.JournalEntryID, m.kind, m.entry())
}
