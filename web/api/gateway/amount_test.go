package gateway

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestAmount checks that an Amount is a JSON string that keeps every digit,
// both ways, as its schema says.
func TestAmount(t *testing.T) {
	tests := map[string]struct {
		amount string
		want   string
	}{
		"a whole amount":                   {amount: "1200", want: `"1200"`},
		"cents":                            {amount: "1200.50", want: `"1200.5"`},
		"a negative amount":                {amount: "-0.01", want: `"-0.01"`},
		"more digits than a float64 keeps": {amount: "12345678901234567.89", want: `"12345678901234567.89"`},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			amount := NewAmount(decimal.RequireFromString(tc.amount))

			encoded, err := json.Marshal(amount)
			require.NoError(t, err)
			require.JSONEq(t, tc.want, string(encoded))
			require.Regexp(t, Amount{}.Schema(nil).Pattern, string(encoded[1:len(encoded)-1]))

			var decoded Amount
			require.NoError(t, json.Unmarshal(encoded, &decoded))
			require.True(t, amount.Equal(decoded.Decimal))
		})
	}
}
