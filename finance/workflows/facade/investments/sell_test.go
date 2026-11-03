package investments

import (
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	"github.com/shopspring/decimal"
)

// TestBuildSell checks that a sale moves the amount less the fee from
// "Investments" to "Cash" and removes the units with no basis.
func TestBuildSell(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	b := &brokerage{currency: euro, cash: cash.ID, investments: investments.ID}

	tests := map[string]struct {
		input    it.SellAssetInput
		proceeds decimal.Decimal
	}{
		"with a fee and a note": {
			input: it.SellAssetInput{
				IdempotencyKey: "8e41", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(4),
				Amount: decimal.RequireFromString("480.00"), Fee: decimal.RequireFromString("3.00"), CurrencyCode: "EUR",
				OccurredAt: occurredAt, Description: "sell to cover",
			},
			proceeds: decimal.NewFromInt(477),
		},
		"of a fraction of a unit, without a fee": {
			input: it.SellAssetInput{
				IdempotencyKey: "8e41", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.RequireFromString("0.5"),
				Amount: decimal.RequireFromString("60.25"), CurrencyCode: "EUR", OccurredAt: occurredAt,
			},
			proceeds: decimal.RequireFromString("60.25"),
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			requireTrade(t, trade{
				kind:               "sell",
				idempotencyKey:     "8e41",
				financialAccountID: ibkr,
				assetID:            7,
				from:               investments.ID,
				to:                 cash.ID,
				amount:             tc.proceeds,
				unitsDelta:         tc.input.Units.Neg(),
				occurredAt:         occurredAt,
				description:        tc.input.Description,
			}, buildSell(&tc.input, b))
		})
	}
}
