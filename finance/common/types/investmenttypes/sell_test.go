package investmenttypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestSellAssetInput_Validate(t *testing.T) {
	valid := func() SellAssetInput {
		return SellAssetInput{
			IdempotencyKey: "8e41", FinancialAccountID: 1, AssetID: 2,
			Units: decimal.NewFromInt(4), Amount: decimal.RequireFromString("480.00"), CurrencyCode: "EUR",
			OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*SellAssetInput)
		wantError string
	}{
		"a sale":                  {mutate: func(*SellAssetInput) {}},
		"a sale with a fee":       {mutate: func(i *SellAssetInput) { i.Fee = decimal.RequireFromString("3.00") }},
		"a sale of a fraction":    {mutate: func(i *SellAssetInput) { i.Units = decimal.RequireFromString("0.125") }},
		"a sale with a note":      {mutate: func(i *SellAssetInput) { i.Description = "sell to cover" }},
		"a blank idempotency key": {mutate: func(i *SellAssetInput) { i.IdempotencyKey = "" }, wantError: "validation error"},
		"no financial account":    {mutate: func(i *SellAssetInput) { i.FinancialAccountID = 0 }, wantError: "validation error"},
		"no asset":                {mutate: func(i *SellAssetInput) { i.AssetID = -1 }, wantError: "validation error"},
		"zero units":              {mutate: func(i *SellAssetInput) { i.Units = decimal.Zero }, wantError: "validation error"},
		"negative units":          {mutate: func(i *SellAssetInput) { i.Units = decimal.NewFromInt(-4) }, wantError: "validation error"},
		"a zero amount":           {mutate: func(i *SellAssetInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":       {mutate: func(i *SellAssetInput) { i.Amount = decimal.NewFromInt(-480) }, wantError: "validation error"},
		"a negative fee":          {mutate: func(i *SellAssetInput) { i.Fee = decimal.NewFromInt(-3) }, wantError: "validation error"},
		"no currency":             {mutate: func(i *SellAssetInput) { i.CurrencyCode = "\t" }, wantError: "validation error"},
		"no occurrence date":      {mutate: func(i *SellAssetInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
		"a fee equal to the amount": {
			mutate:    func(i *SellAssetInput) { i.Fee = decimal.RequireFromString("480") },
			wantError: "invalid parameter: the fee of a sale must be less than its amount",
		},
		"a fee above the amount": {
			mutate:    func(i *SellAssetInput) { i.Fee = decimal.RequireFromString("480.01") },
			wantError: "invalid parameter: the fee of a sale must be less than its amount",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)
			err := input.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
