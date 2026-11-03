package investments

import (
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	"github.com/shopspring/decimal"
)

// TestBuildBuy checks that a buy moves the amount and the fee from "Cash" to
// "Investments" and adds the units at that cost.
func TestBuildBuy(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	b := &brokerage{currency: euro, cash: cash.ID, investments: investments.ID}

	tests := map[string]struct {
		input it.BuyAssetInput
		cost  decimal.Decimal
	}{
		"with a fee and a note": {
			input: it.BuyAssetInput{
				IdempotencyKey: "5d0c", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(10),
				Amount: decimal.RequireFromString("1050.00"), Fee: decimal.RequireFromString("3.00"), CurrencyCode: "EUR",
				OccurredAt: occurredAt, Description: "monthly plan",
			},
			cost: decimal.NewFromInt(1053),
		},
		"of a fraction of a unit, without a fee": {
			input: it.BuyAssetInput{
				IdempotencyKey: "5d0c", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.RequireFromString("0.5"),
				Amount: decimal.RequireFromString("52.50"), CurrencyCode: "EUR", OccurredAt: occurredAt,
			},
			cost: decimal.RequireFromString("52.50"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			requireTrade(t, trade{
				kind:               "buy",
				idempotencyKey:     "5d0c",
				financialAccountID: ibkr,
				assetID:            7,
				from:               cash.ID,
				to:                 investments.ID,
				amount:             tc.cost,
				unitsDelta:         tc.input.Units,
				costBasis:          &tc.cost,
				currencyID:         euro.ID,
				occurredAt:         occurredAt,
				description:        tc.input.Description,
			}, buildBuy(&tc.input, b))
		})
	}
}
