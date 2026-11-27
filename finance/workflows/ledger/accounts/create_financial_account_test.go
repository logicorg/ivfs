package accounts

import (
	"errors"
	"time"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCreateFinancialAccount() {
	tests := []struct {
		name  string
		input act.CreateFinancialAccountInput
		// institution is the account's institution, retrieved only for the
		// stray's name.
		institution *model.Institution
		creates     bool
		// createErr is the error returned by the CreateFinancialAccount
		// activity, if any.
		createErr error
		wantErr   string
	}{
		{
			name:    "an ordinary account",
			input:   act.CreateFinancialAccountInput{InstitutionID: 10, Name: "Revolut EUR"},
			creates: true,
		},
		{
			name:        "the stray's name under another institution",
			input:       act.CreateFinancialAccountInput{InstitutionID: 10, Name: "Untracked accounts"},
			institution: &model.Institution{ID: 10, Name: "Revolut"},
			creates:     true,
		},
		{
			name:      "the activity fails",
			input:     act.CreateFinancialAccountInput{InstitutionID: 10, Name: "Revolut EUR"},
			creates:   true,
			createErr: assert.AnError,
			wantErr:   "create financial account",
		},
		{
			name:        "the stray financial account",
			input:       act.CreateFinancialAccountInput{InstitutionID: 10, Name: "untracked ACCOUNTS"},
			institution: &model.Institution{ID: 10, Name: "Untracked"},
			wantErr:     `"untracked ACCOUNTS" cannot be created: the account "Untracked accounts" of the institution "Untracked" is managed by the ledger; pick another name`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			ia := institutionactivity.Institution{}

			if tc.institution != nil {
				env.OnActivity(ia.Retrieve, mock.Anything, it.RetrieveInstitutionParams{ID: tc.input.InstitutionID}).
					Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil).Once()
			}
			if tc.creates {
				var result *act.CreateFinancialAccountResult
				if tc.createErr == nil {
					result = &act.CreateFinancialAccountResult{FinancialAccountID: 1}
				}
				env.OnActivity(aa.CreateFinancialAccount, mock.Anything, tc.input).Return(result, tc.createErr)
			}

			env.ExecuteWorkflow(CreateFinancialAccount, tc.input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *act.CreateFinancialAccountResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), int64(1), result.FinancialAccountID)
			env.AssertExpectations(s.T())
		})
	}
}

func (s *WorkflowUnitTestSuite) TestCreateFinancialAccountPropagatesInstitutionError() {
	input := act.CreateFinancialAccountInput{InstitutionID: 10, Name: "Untracked accounts"}

	env := s.NewTestWorkflowEnvironment()
	ia := institutionactivity.Institution{}

	env.OnActivity(ia.Retrieve, mock.Anything, it.RetrieveInstitutionParams{ID: 10}).Return(nil, assert.AnError)

	env.ExecuteWorkflow(CreateFinancialAccount, input)

	require.ErrorContains(s.T(), env.GetWorkflowError(), "create financial account")
}

func (s *WorkflowUnitTestSuite) TestCreateFinancialAccountTransientActivityError() {
	input := act.CreateFinancialAccountInput{InstitutionID: 10, Name: "Revolut EUR"}
	okResult := &act.CreateFinancialAccountResult{FinancialAccountID: 1}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.CreateFinancialAccount, mock.Anything, input).Return(nil, errors.New("transient error")).Twice()
	env.OnActivity(aa.CreateFinancialAccount, mock.Anything, input).Return(okResult, nil).After(30 * time.Second)

	env.ExecuteWorkflow(CreateFinancialAccount, input)

	require.NoError(s.T(), env.GetWorkflowError())
	var result *act.CreateFinancialAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), okResult, result)
}
