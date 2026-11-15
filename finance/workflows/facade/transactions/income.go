package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RecordIncome records money from an income category arriving in a balance:
// "Income" in the balance's currency is credited, with the category on that
// posting, and the balance debited (ADR 0008). A cashback paid onto a card
// debits a liability, so the amount owed shrinks. It fails with
// ErrInvalidParameter, writing nothing, when the balance is not a balance or
// the category is not an income one. A repeated idempotency key returns the
// entry first posted with it.
func RecordIncome(ctx workflow.Context, input tt.RecordIncomeInput) (*tt.RecordResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	m, err := incomeMovement(ctx, input)
	if err != nil {
		return nil, err
	}
	return post(ctx, m)
}

// incomeMovement checks that the balance and the category can play their
// roles in an income and returns the movement it is, for recording it or for
// updating an entry to it.
func incomeMovement(ctx workflow.Context, input tt.RecordIncomeInput) (movement, error) {
	balance, income, err := categorized(ctx, input.BalanceLedgerAccountID, input.CategoryID, model.LedgerAccountIncome)
	if err != nil {
		return movement{}, err
	}

	return movement{
		kind:           "income",
		idempotencyKey: input.IdempotencyKey,
		from:           income,
		fromCategory:   input.CategoryID,
		to:             balance.ID,
		amount:         input.Amount,
		occurredAt:     input.OccurredAt,
		description:    input.Description,
	}, nil
}
