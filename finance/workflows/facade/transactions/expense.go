package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RecordExpense records money paid from a balance for an expense category: the
// balance is credited and the category debited (ADR 0008). Paying by card
// credits a liability, so the amount owed grows. It fails with
// ErrInvalidParameter, writing nothing, when the balance is not a balance, the
// category is not an expense one, or the two are in different currencies. A
// repeated idempotency key returns the entry first posted with it.
func RecordExpense(ctx workflow.Context, input tt.RecordExpenseInput) (*tt.RecordResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	accounts, err := retrieveLedgerAccounts(ctx, input.BalanceLedgerAccountID, input.CategoryLedgerAccountID)
	if err != nil {
		return nil, err
	}
	balance, category := accounts[0], accounts[1]
	if err := checkBalance(balance); err != nil {
		return nil, err
	}
	if err := checkCategory(category, model.LedgerAccountExpense); err != nil {
		return nil, err
	}
	if err := checkSameCurrency(balance, category); err != nil {
		return nil, err
	}

	return post(ctx, movement{
		kind:           "expense",
		idempotencyKey: input.IdempotencyKey,
		from:           balance.ID,
		to:             category.ID,
		amount:         input.Amount,
		occurredAt:     input.OccurredAt,
		description:    input.Description,
	})
}
