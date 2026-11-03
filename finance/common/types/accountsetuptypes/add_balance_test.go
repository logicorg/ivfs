package accountsetuptypes

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestAddBalanceInput_Validate(t *testing.T) {
	tests := map[string]struct {
		input     AddBalanceInput
		wantError bool
	}{
		"a balance with an opening balance": {
			input: AddBalanceInput{FinancialAccountID: 1, Name: "Savings vault", CurrencyCode: "EUR", OpeningBalance: decimal.NewFromInt(500)},
		},
		"no financial account":  {input: AddBalanceInput{Name: "Savings vault", CurrencyCode: "EUR"}, wantError: true},
		"a negative account id": {input: AddBalanceInput{FinancialAccountID: -1, Name: "Savings vault", CurrencyCode: "EUR"}, wantError: true},
		"a blank name":          {input: AddBalanceInput{FinancialAccountID: 1, Name: " ", CurrencyCode: "EUR"}, wantError: true},
		"a blank currency code": {input: AddBalanceInput{FinancialAccountID: 1, Name: "Savings vault"}, wantError: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantError {
				require.ErrorContains(t, err, "validation error")
				return
			}
			require.NoError(t, err)
		})
	}
}
