package ledgertypes

import (
	"testing"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRegisterInvestmentTradeInput_Validate(t *testing.T) {
	valid := func() RegisterInvestmentTradeInput {
		return RegisterInvestmentTradeInput{
			FinancialAccountID: 1,
			AssetID:            1,
			UnitsDelta:         decimal.NewFromInt(3),
			Entry: PostJournalEntryInput{
				IdempotencyKey: "buy-aapl",
				OccurredAt:     time.Now(),
				Postings: []PostingInput{
					{LedgerAccountID: 1, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					{LedgerAccountID: 2, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				},
			},
		}
	}

	const (
		accepted   = ""
		constraint = "cue"
		rule       = "rule"
	)

	tests := map[string]struct {
		mutate func(*RegisterInvestmentTradeInput)
		// failure is how the input is rejected: by a CUE constraint, which is a
		// ValidationError, or by a rule CUE cannot express, which is an
		// ErrInvalidParameter.
		failure string
	}{
		"a buy": {mutate: func(*RegisterInvestmentTradeInput) {}},
		"a sale": {mutate: func(i *RegisterInvestmentTradeInput) {
			i.UnitsDelta = decimal.NewFromInt(-3)
		}},
		"fractional units": {mutate: func(i *RegisterInvestmentTradeInput) {
			i.UnitsDelta = decimal.RequireFromString("0.000001")
		}},
		"a cost basis with its currency": {mutate: func(i *RegisterInvestmentTradeInput) {
			i.CostBasisAmount, i.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(30)), ptr.Int64(1)
		}},
		"a negative cost basis, as a sale removes basis": {mutate: func(i *RegisterInvestmentTradeInput) {
			i.UnitsDelta = decimal.NewFromInt(-3)
			i.CostBasisAmount, i.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(-30)), ptr.Int64(1)
		}},
		"no financial account": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.FinancialAccountID = 0
		}},
		"no asset": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.AssetID = 0
		}},
		"zero units": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.UnitsDelta = decimal.Zero
		}},
		"zero units written with decimals": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.UnitsDelta = decimal.RequireFromString("0.00")
		}},
		"no idempotency key": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.Entry.IdempotencyKey = ""
		}},
		"a cost basis currency that is not an id": {failure: constraint, mutate: func(i *RegisterInvestmentTradeInput) {
			i.CostBasisAmount, i.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(30)), ptr.Int64(0)
		}},
		"a cost basis without a currency": {failure: rule, mutate: func(i *RegisterInvestmentTradeInput) {
			i.CostBasisAmount = ptr.NewPtr(decimal.NewFromInt(30))
		}},
		"a cost basis currency without an amount": {failure: rule, mutate: func(i *RegisterInvestmentTradeInput) {
			i.CostBasisCurrencyID = ptr.Int64(1)
		}},
		"an entry with a single posting": {failure: rule, mutate: func(i *RegisterInvestmentTradeInput) {
			i.Entry.Postings = i.Entry.Postings[:1]
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)

			err := input.Validate()

			switch tc.failure {
			case accepted:
				require.NoError(t, err)
			case constraint:
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
			case rule:
				require.ErrorAs(t, err, &internalerrors.ErrInvalidParameter{})
			}
		})
	}
}
