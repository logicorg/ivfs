package investments

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

// RecordInvestment records one investment transaction of any kind (ADR 0009):
// it looks up the balances and categories the kind's facts name, has the
// kind's builder turn the facts into an entry and a movement, and writes both
// in one transaction. The builders do no I/O, so a new kind is a builder and a
// case here rather than a workflow. A sale of more units than are held is
// written, and its negative holding comes back as a warning (ADR 0010). It
// fails with ErrInvalidParameter, writing nothing, when the account has no
// "Cash" and "Investments" balances in the currency, or a vest's category is
// not of its kind or currency. A repeated idempotency key returns the entry and
// movement first written with it.
func RecordInvestment(ctx workflow.Context, input it.RecordInvestmentInput) (*it.TradeResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	switch input.Kind {
	case it.KindBuy:
		b, err := retrieveBrokerage(ctx, input.Buy.FinancialAccountID, input.Buy.CurrencyCode)
		if err != nil {
			return nil, err
		}
		return register(ctx, buildBuy(input.Buy, b))
	case it.KindSell:
		b, err := retrieveBrokerage(ctx, input.Sell.FinancialAccountID, input.Sell.CurrencyCode)
		if err != nil {
			return nil, err
		}
		return register(ctx, buildSell(input.Sell, b))
	case it.KindVest:
		b, err := retrieveBrokerage(ctx, input.Vest.FinancialAccountID, input.Vest.CurrencyCode)
		if err != nil {
			return nil, err
		}
		categories, err := retrieveVestCategories(ctx, input.Vest, b.currency)
		if err != nil {
			return nil, err
		}
		return registerVest(ctx, buildVest(input.Vest, b, categories))
	}
	// Validate refuses any other kind, so this is a kind added there and not
	// here.
	return nil, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("kind %q cannot be recorded yet", input.Kind)}
}
