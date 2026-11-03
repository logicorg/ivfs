package accounts

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestMoveLedgerAccount() {
	tests := []struct {
		name     string
		account  model.LedgerAccount
		parentID *int64
		moves    bool
		wantErr  string
	}{
		{
			name:     "an ordinary account under a parent",
			account:  model.LedgerAccount{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense},
			parentID: ptr.NewPtr(int64(8)),
			moves:    true,
		},
		{
			name:    "an ordinary account to the top level",
			account: model.LedgerAccount{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense, ParentID: ptr.NewPtr(int64(8))},
			moves:   true,
		},
		{
			name:     "a shared account",
			account:  model.LedgerAccount{ID: 7, Name: "Opening balances", Kind: model.LedgerAccountEquity},
			parentID: ptr.NewPtr(int64(8)),
			wantErr:  `"Opening balances" (equity) is managed by the ledger and cannot be moved`,
		},
		{
			name:    "a child of a stray name to the top level",
			account: model.LedgerAccount{ID: 7, Name: "Untracked accounts", Kind: model.LedgerAccountEquity, ParentID: ptr.NewPtr(int64(8))},
			wantErr: `"Untracked accounts" cannot be moved to the top level: a top-level equity account named "Untracked accounts" is managed by the ledger`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			input := lt.MoveLedgerAccountInput{LedgerAccountID: tc.account.ID, ParentID: tc.parentID}

			env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: tc.account.ID}).
				Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: &tc.account}, nil)
			if tc.moves {
				env.OnActivity(aa.MoveLedgerAccount, mock.Anything, input).
					Return(&lt.LedgerAccountMutationResult{LedgerAccountID: tc.account.ID}, nil).Once()
			}

			env.ExecuteWorkflow(MoveLedgerAccount, input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}
