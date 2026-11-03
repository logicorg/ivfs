package investmenttypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestBuyAssetInput_Validate(t *testing.T) {
	valid := func() BuyAssetInput {
		return BuyAssetInput{
			IdempotencyKey: "5d0c", FinancialAccountID: 1, AssetID: 2,
			Units: decimal.NewFromInt(10), Amount: decimal.RequireFromString("1050.00"), CurrencyCode: "EUR",
			OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*BuyAssetInput)
		wantError string
	}{
		"a buy":                   {mutate: func(*BuyAssetInput) {}},
		"a buy with a fee":        {mutate: func(i *BuyAssetInput) { i.Fee = decimal.RequireFromString("3.00") }},
		"a buy of a fraction":     {mutate: func(i *BuyAssetInput) { i.Units = decimal.RequireFromString("0.125") }},
		"a buy with a note":       {mutate: func(i *BuyAssetInput) { i.Description = "monthly plan" }},
		"a blank idempotency key": {mutate: func(i *BuyAssetInput) { i.IdempotencyKey = " " }, wantError: "validation error"},
		"no financial account":    {mutate: func(i *BuyAssetInput) { i.FinancialAccountID = 0 }, wantError: "validation error"},
		"no asset":                {mutate: func(i *BuyAssetInput) { i.AssetID = 0 }, wantError: "validation error"},
		"zero units":              {mutate: func(i *BuyAssetInput) { i.Units = decimal.Zero }, wantError: "validation error"},
		"negative units":          {mutate: func(i *BuyAssetInput) { i.Units = decimal.NewFromInt(-10) }, wantError: "validation error"},
		"a zero amount":           {mutate: func(i *BuyAssetInput) { i.Amount = decimal.RequireFromString("0.00") }, wantError: "validation error"},
		"a negative amount":       {mutate: func(i *BuyAssetInput) { i.Amount = decimal.NewFromInt(-1050) }, wantError: "validation error"},
		"a negative fee":          {mutate: func(i *BuyAssetInput) { i.Fee = decimal.NewFromInt(-3) }, wantError: "validation error"},
		"no currency":             {mutate: func(i *BuyAssetInput) { i.CurrencyCode = "" }, wantError: "validation error"},
		"no occurrence date":      {mutate: func(i *BuyAssetInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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
