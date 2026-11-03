package accountsetuptypes

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestOpenAccountInput_Validate(t *testing.T) {
	eur := []CurrencyBalance{{Code: "EUR"}}

	tests := map[string]struct {
		input     OpenAccountInput
		wantError string
	}{
		"a bank account": {
			input: OpenAccountInput{Type: AccountTypeBank, InstitutionName: "Revolut", Currencies: eur},
		},
		"a brokerage with two currencies and opening balances": {
			input: OpenAccountInput{Type: AccountTypeBrokerage, InstitutionName: "IBKR", Currencies: []CurrencyBalance{
				{Code: "EUR", OpeningBalance: decimal.NewFromInt(5000)}, {Code: "USD"},
			}},
		},
		"a cash account without an institution": {
			input: OpenAccountInput{Type: AccountTypeCash, Currencies: eur},
		},
		"an unknown type": {
			input:     OpenAccountInput{Type: "loan", InstitutionName: "Bank", Currencies: eur},
			wantError: `unknown account type "loan" (must be one of ["bank" "card" "brokerage" "cash"])`,
		},
		"no type": {
			input:     OpenAccountInput{InstitutionName: "Bank", Currencies: eur},
			wantError: `unknown account type ""`,
		},
		"no currencies": {
			input:     OpenAccountInput{Type: AccountTypeBank, InstitutionName: "Revolut"},
			wantError: "at least one currency is required",
		},
		"a blank currency code": {
			input:     OpenAccountInput{Type: AccountTypeBank, InstitutionName: "Revolut", Currencies: []CurrencyBalance{{Code: " "}}},
			wantError: "validation error",
		},
		"the same currency twice, in another casing": {
			input:     OpenAccountInput{Type: AccountTypeBank, InstitutionName: "Revolut", Currencies: []CurrencyBalance{{Code: "EUR"}, {Code: "eur"}}},
			wantError: "currency EUR is given more than once",
		},
		"no institution": {
			input:     OpenAccountInput{Type: AccountTypeBank, Currencies: eur},
			wantError: "the institution is required for a bank account",
		},
		"a blank institution": {
			input:     OpenAccountInput{Type: AccountTypeCard, InstitutionName: "  ", Currencies: eur},
			wantError: "the institution is required for a card account",
		},
		"an institution for a cash account": {
			input:     OpenAccountInput{Type: AccountTypeCash, InstitutionName: "Wallet Bank", Currencies: eur},
			wantError: `a cash account belongs to "Physical cash"`,
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
