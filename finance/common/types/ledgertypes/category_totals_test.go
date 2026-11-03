package ledgertypes

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCategoryTotalsParams_Validate(t *testing.T) {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)

	tt := []struct {
		name      string
		params    CategoryTotalsParams
		wantError string
	}{
		{name: "a month", params: CategoryTotalsParams{From: january, To: february}},
		{name: "no start", params: CategoryTotalsParams{To: february}, wantError: "missing required parameter"},
		{name: "no end", params: CategoryTotalsParams{From: january}, wantError: "missing required parameter"},
		{name: "an empty period", params: CategoryTotalsParams{From: january, To: january}, wantError: "invalid parameter"},
		{name: "the end before the start", params: CategoryTotalsParams{From: february, To: january}, wantError: "invalid parameter"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}
