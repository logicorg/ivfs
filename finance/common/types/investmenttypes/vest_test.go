package investmenttypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestVestInput_Validate(t *testing.T) {
	valid := func() VestInput {
		return VestInput{
			IdempotencyKey: "a71f", FinancialAccountID: 1, AssetID: 2,
			UnitsVested: decimal.NewFromInt(100), FairMarketValue: decimal.RequireFromString("10.00"), CurrencyCode: "USD",
			IncomeCategoryLedgerAccountID: 3, OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}
	withheld := func(units string) func(*VestInput) {
		return func(i *VestInput) {
			i.UnitsWithheld = decimal.RequireFromString(units)
			i.TaxCategoryLedgerAccountID = 4
		}
	}

	tests := map[string]struct {
		mutate    func(*VestInput)
		wantError string
	}{
		"a vest":                            {mutate: func(*VestInput) {}},
		"a vest with units withheld":        {mutate: withheld("40")},
		"a vest of a fraction":              {mutate: func(i *VestInput) { i.UnitsVested = decimal.RequireFromString("12.5") }},
		"a vest with a tax category unused": {mutate: func(i *VestInput) { i.TaxCategoryLedgerAccountID = 4 }},
		"a vest with a note":                {mutate: func(i *VestInput) { i.Description = "Q3 vest" }},
		"a blank idempotency key":           {mutate: func(i *VestInput) { i.IdempotencyKey = " " }, wantError: "validation error"},
		"no financial account":              {mutate: func(i *VestInput) { i.FinancialAccountID = 0 }, wantError: "validation error"},
		"no asset":                          {mutate: func(i *VestInput) { i.AssetID = 0 }, wantError: "validation error"},
		"zero units vested":                 {mutate: func(i *VestInput) { i.UnitsVested = decimal.Zero }, wantError: "validation error"},
		"negative units vested":             {mutate: func(i *VestInput) { i.UnitsVested = decimal.NewFromInt(-100) }, wantError: "validation error"},
		"negative units withheld":           {mutate: withheld("-40"), wantError: "validation error"},
		"a zero fair market value":          {mutate: func(i *VestInput) { i.FairMarketValue = decimal.Zero }, wantError: "validation error"},
		"a negative fair market value":      {mutate: func(i *VestInput) { i.FairMarketValue = decimal.NewFromInt(-10) }, wantError: "validation error"},
		"no currency":                       {mutate: func(i *VestInput) { i.CurrencyCode = "" }, wantError: "validation error"},
		"no income category":                {mutate: func(i *VestInput) { i.IncomeCategoryLedgerAccountID = 0 }, wantError: "validation error"},
		"a negative tax category":           {mutate: func(i *VestInput) { i.TaxCategoryLedgerAccountID = -4 }, wantError: "validation error"},
		"no occurrence date":                {mutate: func(i *VestInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
		"units withheld without a category": {
			mutate:    func(i *VestInput) { i.UnitsWithheld = decimal.NewFromInt(40) },
			wantError: "missing required parameters: TaxCategoryLedgerAccountID",
		},
		"every unit withheld": {
			mutate:    withheld("100"),
			wantError: "invalid parameter: the units withheld for tax must be fewer than the units vested",
		},
		"more units withheld than vested": {
			mutate:    withheld("100.5"),
			wantError: "invalid parameter: the units withheld for tax must be fewer than the units vested",
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
