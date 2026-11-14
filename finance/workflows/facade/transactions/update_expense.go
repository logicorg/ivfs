package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"go.temporal.io/sdk/workflow"
)

// UpdateExpense makes a journal entry an expense with the given fields,
// whatever it was before, with the checks of RecordExpense. The entry keeps its
// id, and loses any position movement it had, so the holdings left negative
// are returned as warnings. It fails with ErrInvalidParameter, writing
// nothing, for what RecordExpense refuses, and with ErrNotFound when the entry
// does not exist.
func UpdateExpense(ctx workflow.Context, input tt.UpdateExpenseInput) (*tt.UpdateResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	m, err := expenseMovement(ctx, input.Expense())
	if err != nil {
		return nil, err
	}
	return replace(ctx, input.JournalEntryID, m.kind, m.entry())
}
