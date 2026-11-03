package accounts

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCloseLedgerAccount() {
	tests := []struct {
		name    string
		account model.LedgerAccount
		// closes sets whether the workflow should call the CloseLedgerAccount activity.
		closes bool
		// closeErr is the error returned by the CloseLedgerAccount activity (if any – defaults to nil)
		closeErr error
		wantErr  string
	}{
		{
			name:    "an ordinary account",
			account: model.LedgerAccount{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense},
			closes:  true,
		},
		{
			name:    "a balance with a shared name",
			account: model.LedgerAccount{ID: 7, Name: "FX clearing", Kind: model.LedgerAccountAsset, FinancialAccountID: ptr.NewPtr(int64(3))},
			closes:  true,
		},
		{
			name:     "the activity fails",
			account:  model.LedgerAccount{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense},
			closeErr: assert.AnError,
			closes:   true,
			wantErr:  "close ledger account",
		},
		{
			name:    "a shared account",
			account: model.LedgerAccount{ID: 7, Name: "FX clearing", Kind: model.LedgerAccountAsset},
			wantErr: `"FX clearing" (asset) is managed by the ledger and cannot be closed`,
		},
		{
			name:    "a stray account",
			account: model.LedgerAccount{ID: 7, Name: "Uncategorized", Kind: model.LedgerAccountExpense},
			wantErr: `"Uncategorized" (expense) is managed by the ledger and cannot be closed`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			input := lt.CloseLedgerAccountInput{LedgerAccountID: tc.account.ID, ClosedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

			env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.account.ID}).
				Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &tc.account}, nil)

			if tc.closes {
				env.OnActivity(aa.CloseLedgerAccount, mock.Anything, input).Return(tc.closeErr)
			}

			env.ExecuteWorkflow(CloseLedgerAccount, input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}
