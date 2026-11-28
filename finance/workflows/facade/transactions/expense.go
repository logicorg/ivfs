package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RecordExpense records money paid from a balance for an expense category: the
// balance is credited and "Expenses" in its currency debited, with the
// category on that posting (ADR 0008). A payment spent on several things gives
// lines instead of a category, and "Expenses" is debited once per line, in the
// one entry. Paying by card credits a liability, so the amount owed grows. It fails with ErrInvalidParameter, writing nothing,
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

// expenseMovement checks that the balance and the categories can play their
// roles in an expense and returns the movement it is, for recording it or for
// updating an entry to it. With lines, the first is the movement's own debit
// and the others its splits.
func expenseMovement(ctx workflow.Context, input tt.RecordExpenseInput) (movement, error) {
	categoryID, others := input.CategoryID, input.Lines
	if len(input.Lines) > 0 {
		categoryID, others = input.Lines[0].CategoryID, input.Lines[1:]
	}
	balance, expenses, err := categorized(ctx, input.BalanceLedgerAccountID, categoryID, model.LedgerAccountExpense)
	if err != nil {
		return movement{}, err
	}
	splits, err := expenseSplits(ctx, others, balance.CurrencyID)
	if err != nil {
		return movement{}, err
	}

	return movement{
		kind:           "expense",
		idempotencyKey: input.IdempotencyKey,
		from:           balance.ID,
		to:             expenses,
		toCategory:     categoryID,
		splits:         splits,
		amount:         input.Amount,
		occurredAt:     input.OccurredAt,
		description:    input.Description,
		tagIDs:         input.TagIDs,
	}, nil
}
