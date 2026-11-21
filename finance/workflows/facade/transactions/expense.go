package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RecordExpense records money paid from a balance for an expense category: the
// balance is credited and "Expenses" in its currency debited, with the
// category on that posting (ADR 0008). Paying by card credits a liability, so
// the amount owed grows. It fails with ErrInvalidParameter, writing nothing,
// when the balance is not a balance or the category is not an expense one. A
// repeated idempotency key returns the entry first posted with it.
func RecordExpense(ctx workflow.Context, input tt.RecordExpenseInput) (*tt.RecordResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	m, err := expenseMovement(ctx, input)
	if err != nil {
		return nil, err
	}
	return post(ctx, m)
}

// expenseMovement checks that the balance and the category can play their
// roles in an expense and returns the movement it is, for recording it or for
// updating an entry to it.
func expenseMovement(ctx workflow.Context, input tt.RecordExpenseInput) (movement, error) {
	balance, expenses, err := categorized(ctx, input.BalanceLedgerAccountID, input.CategoryID, model.LedgerAccountExpense)
	if err != nil {
		return movement{}, err
	}

	return movement{
		kind:           "expense",
		idempotencyKey: input.IdempotencyKey,
		from:           balance.ID,
		to:             expenses,
		toCategory:     input.CategoryID,
		amount:         input.Amount,
		occurredAt:     input.OccurredAt,
		description:    input.Description,
		tagIDs:         input.TagIDs,
	}, nil
}
