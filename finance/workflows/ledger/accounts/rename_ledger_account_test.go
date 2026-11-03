package accounts

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRenameLedgerAccount() {
	tests := []struct {
		name    string
		account model.LedgerAccount
		newName string
		renames bool
		wantErr string
	}{
		{
			name:    "an ordinary account",
			account: model.LedgerAccount{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense},
			newName: "Food",
			renames: true,
		},
		{
			name:    "a balance onto a shared name",
			account: model.LedgerAccount{ID: 7, Name: "Cash", Kind: model.LedgerAccountAsset, FinancialAccountID: ptr.NewPtr(int64(3))},
			newName: "FX clearing",
			renames: true,
		},
		{
			name:    "a shared account",
			account: model.LedgerAccount{ID: 7, Name: "FX clearing", Kind: model.LedgerAccountAsset},
			newName: "Clearing",
			wantErr: `"FX clearing" (asset) is managed by the ledger and cannot be renamed`,
		},
		{
			name:    "a stray account",
			account: model.LedgerAccount{ID: 7, Name: "Uncategorized", Kind: model.LedgerAccountIncome},
			newName: "Other income",
			wantErr: `"Uncategorized" (income) is managed by the ledger and cannot be renamed`,
		},
		{
			name:    "a top-level account onto a shared name",
			account: model.LedgerAccount{ID: 7, Name: "Starting point", Kind: model.LedgerAccountEquity},
			newName: "opening balances",
			wantErr: `"Starting point" cannot be renamed: a top-level equity account named "opening balances" is managed by the ledger`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			input := lt.RenameLedgerAccountInput{LedgerAccountID: tc.account.ID, Name: tc.newName}

			env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.account.ID}).
				Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &tc.account}, nil)
			if tc.renames {
				env.OnActivity(aa.RenameLedgerAccount, mock.Anything, input).
					Return(&lt.LedgerAccountMutationResult{LedgerAccountID: tc.account.ID}, nil).Once()
			}

			env.ExecuteWorkflow(RenameLedgerAccount, input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}

func (s *WorkflowUnitTestSuite) TestRenameLedgerAccountPropagatesRetrieveError() {
	input := lt.RenameLedgerAccountInput{LedgerAccountID: 7, Name: "Food"}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: 7}).Return(nil, assert.AnError)

	env.ExecuteWorkflow(RenameLedgerAccount, input)

	require.ErrorContains(s.T(), env.GetWorkflowError(), "rename ledger account")
}
