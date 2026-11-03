package currencytypes

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func Test_InsertCurrencyParams_Validate(t *testing.T) {
	tests := map[string]struct {
		input     InsertCurrencyParams
		wantError string
	}{
		"euro": {
			input: InsertCurrencyParams{
				Code:      "EUR",
				Name:      "Euro",
				Symbol:    "€",
				Precision: ptr.NewPtr(2),
			},
		},
		"euro-no-symbol": {
			input: InsertCurrencyParams{
				Code:      "EUR",
				Name:      "Euro",
				Precision: ptr.NewPtr(2),
			},
		},
		"euro-no-precision": {
			input: InsertCurrencyParams{
				Code:   "EUR",
				Name:   "Euro",
				Symbol: "€",
			},
		},
		"euro-no-symbol-no-precision": {
			input: InsertCurrencyParams{
				Code: "EUR",
				Name: "Euro",
			},
		},
		"cad": {
			input: InsertCurrencyParams{
				Code:      "CAD",
				Name:      "Canadian Dollar",
				Symbol:    "$",
				Precision: ptr.NewPtr(2),
			},
		},
		"cad-diff-symbol": {
			input: InsertCurrencyParams{
				Code:      "CAD",
				Name:      "Canadian Dollar",
				Symbol:    "C$",
				Precision: ptr.NewPtr(2),
			},
		},
		"no-code": {
			input:     InsertCurrencyParams{Name: "Euro"},
			wantError: "missing required parameters: Code",
		},
		"blank-name": {
			input:     InsertCurrencyParams{Code: "EUR", Name: " "},
			wantError: "missing required parameters: Name",
		},
		"neither-code-nor-name": {
			input:     InsertCurrencyParams{},
			wantError: "missing required parameters: Code, Name",
		},
		"invalid-code": {
			input: InsertCurrencyParams{
				Code:      "EURO",
				Name:      "Euro",
				Symbol:    "€",
				Precision: ptr.NewPtr(2),
			},
			wantError: "validation error",
		},
		"invalid-name": {
			input: InsertCurrencyParams{
				Code:      "EUR",
				Name:      "Euro1",
				Symbol:    "€",
				Precision: ptr.NewPtr(2),
			},
			wantError: "validation error",
		},
		"invalid-symbol": {
			input: InsertCurrencyParams{
				Code:      "EUR",
				Name:      "Euro",
				Symbol:    "€€€€€€",
				Precision: ptr.NewPtr(2),
			},
			wantError: "validation error",
		},
		"invalid-precision": {
			input: InsertCurrencyParams{
				Code:      "EUR",
				Name:      "Euro",
				Symbol:    "€",
				Precision: ptr.NewPtr(-1),
			},
			wantError: "validation error",
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
