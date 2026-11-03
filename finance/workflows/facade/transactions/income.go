package transactions

import (
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RecordIncome records money from an income category arriving in a balance:
// the category is credited and the balance debited (ADR 0008). A cashback paid
// onto a card debits a liability, so the amount owed shrinks. It fails with
// ErrInvalidParameter, writing nothing, when the balance is not a balance, the
// category is not an income one, or the two are in different currencies. A
// repeated idempotency key returns the entry first posted with it.
func RecordIncome(ctx workflow.Context, input tt.RecordIncomeInput) (*tt.RecordResult, error) {
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
	if err := checkCategory(category, model.LedgerAccountIncome); err != nil {
		return nil, err
	}
	if err := checkSameCurrency(balance, category); err != nil {
		return nil, err
	}

	return post(ctx, movement{
		kind:           "income",
		idempotencyKey: input.IdempotencyKey,
		from:           category.ID,
		to:             balance.ID,
		amount:         input.Amount,
		occurredAt:     input.OccurredAt,
		description:    input.Description,
	})
}
