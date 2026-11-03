package ledgertypes

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/require"
)

func TestListLedgerAccountsParams_Validate(t *testing.T) {
	tt := []struct {
		name      string
		params    ListLedgerAccountsParams
		wantError bool
	}{
		{
			name:      "empty params",
			params:    ListLedgerAccountsParams{},
			wantError: false,
		},
		{
			name: "valid params",
			params: ListLedgerAccountsParams{
				FinancialAccountID: ptr.Int64(1),
				ParentID:           ptr.Int64(2),
				CurrencyID:         ptr.Int64(3),
				NameContains:       "test",
				Kinds:              []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense},
				IncludeClosed:      true,
			},
			wantError: false,
		},
		{
			name: "invalid FinancialAccountID",
			params: ListLedgerAccountsParams{
				FinancialAccountID: ptr.Int64(-1),
			},
			wantError: true,
		},
		{
			name: "invalid ParentID",
			params: ListLedgerAccountsParams{
				ParentID: ptr.Int64(-1),
			},
			wantError: true,
		},
		{
			name: "invalid CurrencyID",
			params: ListLedgerAccountsParams{
				CurrencyID: ptr.Int64(-1),
			},
			wantError: true,
		},
		{
			name: "invalid kind",
			params: ListLedgerAccountsParams{
				Kinds: []model.LedgerAccountKind{"invalid-kind"},
			},
			wantError: true,
		},
		{
			name: "valid kind",
			params: ListLedgerAccountsParams{
				Kinds: []model.LedgerAccountKind{model.LedgerAccountIncome},
			},
			wantError: false,
		},
		{
			name: "invalid (zero) FinancialAccountID",
			params: ListLedgerAccountsParams{
				FinancialAccountID: ptr.Int64(0),
			},
			wantError: true,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
