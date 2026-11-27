package accounts

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
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
			account: model.LedgerAccount{ID: 7, Name: "Income", Kind: model.LedgerAccountIncome},
			newName: "Other income",
			wantErr: `"Income" (income) is managed by the ledger and cannot be renamed`,
		},
		{
			name:    "an account of no financial account onto a shared name",
			account: model.LedgerAccount{ID: 7, Name: "Starting point", Kind: model.LedgerAccountEquity},
			newName: "opening balances",
			wantErr: `"Starting point" cannot be renamed: the account named "opening balances", of no financial account, is managed by the ledger`,
		},
		{
			name:    "an account of no financial account onto a shared name of another kind",
			account: model.LedgerAccount{ID: 7, Name: "Starting point", Kind: model.LedgerAccountEquity},
			newName: "FX clearing",
			wantErr: `"Starting point" cannot be renamed: the account named "FX clearing", of no financial account, is managed by the ledger`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			input := act.RenameLedgerAccountInput{LedgerAccountID: tc.account.ID, Name: tc.newName}

			env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: tc.account.ID}).
				Return(&act.RetrieveLedgerAccountResult{LedgerAccount: &tc.account}, nil)
			if tc.renames {
				env.OnActivity(aa.RenameLedgerAccount, mock.Anything, input).
					Return(&act.LedgerAccountMutationResult{LedgerAccountID: tc.account.ID}, nil).Once()
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
	input := act.RenameLedgerAccountInput{LedgerAccountID: 7, Name: "Food"}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: 7}).Return(nil, assert.AnError)

	env.ExecuteWorkflow(RenameLedgerAccount, input)

	require.ErrorContains(s.T(), env.GetWorkflowError(), "rename ledger account")
}
