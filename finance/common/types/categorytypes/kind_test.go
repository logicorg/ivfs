package categorytypes

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

func TestCheckKind(t *testing.T) {
	tests := []struct {
		name      string
		category  model.Category
		kind      model.LedgerAccountKind
		wantError string
	}{
		{name: "an expense category booked as an expense", category: model.Category{Name: "Groceries", Kind: model.LedgerAccountExpense}, kind: model.LedgerAccountExpense},
		{name: "an income category booked as an income", category: model.Category{Name: "Salary", Kind: model.LedgerAccountIncome}, kind: model.LedgerAccountIncome},
		{
			name:      "an expense category booked as an income",
			category:  model.Category{Name: "Groceries", Kind: model.LedgerAccountExpense},
			kind:      model.LedgerAccountIncome,
			wantError: `invalid parameter: "Groceries" is an expense category, not an income one`,
		},
		{
			name:      "an income category booked as an expense",
			category:  model.Category{Name: "Salary", Kind: model.LedgerAccountIncome},
			kind:      model.LedgerAccountExpense,
			wantError: `invalid parameter: "Salary" is an income category, not an expense one`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckKind(&tc.category, tc.kind)
			if tc.wantError == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, tc.wantError)
		})
	}
}
