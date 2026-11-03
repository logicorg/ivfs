package accounts

import (
	"time"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCloseFinancialAccount() {
	tests := []struct {
		name    string
		account model.FinancialAccount
		// institution is the account's institution, retrieved only for the
		// stray's name.
		institution *model.Institution
		closes      bool
		// closeErr is the error returned by the CloseFinancialAccount
		// activity, if any.
		closeErr error
		wantErr  string
	}{
		{
			name:    "an ordinary account",
			account: model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Revolut EUR"},
			closes:  true,
		},
		{
			name:        "the stray's name under another institution",
			account:     model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Untracked accounts"},
			institution: &model.Institution{ID: 10, Name: "Revolut"},
			closes:      true,
		},
		{
			name:     "the activity fails",
			account:  model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Revolut EUR"},
			closes:   true,
			closeErr: assert.AnError,
			wantErr:  "close financial account",
		},
		{
			name:        "the stray financial account",
			account:     model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Untracked accounts"},
			institution: &model.Institution{ID: 10, Name: "Untracked"},
			wantErr:     `"Untracked accounts" of the institution "Untracked" is managed by the ledger and cannot be closed`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			ia := institutionactivity.Institution{}
			input := lt.CloseFinancialAccountInput{FinancialAccountID: tc.account.ID, ClosedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

			env.OnActivity(aa.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: tc.account.ID}).
				Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: &tc.account}, nil)
			if tc.institution != nil {
				env.OnActivity(ia.Retrieve, mock.Anything, it.RetrieveInstitutionParams{ID: tc.account.InstitutionID}).
					Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil).Once()
			}
			if tc.closes {
				env.OnActivity(aa.CloseFinancialAccount, mock.Anything, input).Return(tc.closeErr).Once()
			}

			env.ExecuteWorkflow(CloseFinancialAccount, input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}
