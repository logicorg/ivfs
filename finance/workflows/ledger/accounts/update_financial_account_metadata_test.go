package accounts

import (
	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestUpdateFinancialAccountMetadata() {
	tests := []struct {
		name    string
		account model.FinancialAccount
		newName *string
		// institution is the account's institution, retrieved only when the
		// account has, or is renamed to, the stray's name.
		institution *model.Institution
		updates     bool
		wantErr     string
	}{
		{
			name:    "a rename",
			account: model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Revolut EUR"},
			newName: ptr.NewPtr("Revolut euro"),
			updates: true,
		},
		{
			name:    "an external ID change",
			account: model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Revolut EUR"},
			updates: true,
		},
		{
			name:        "a rename onto the stray's name under another institution",
			account:     model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Revolut EUR"},
			newName:     ptr.NewPtr("Untracked accounts"),
			institution: &model.Institution{ID: 10, Name: "Revolut"},
			updates:     true,
		},
		{
			name:        "the stray financial account",
			account:     model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Untracked accounts"},
			newName:     ptr.NewPtr("Lost and found"),
			institution: &model.Institution{ID: 10, Name: "Untracked"},
			wantErr:     `"Untracked accounts" of the institution "Untracked" is managed by the ledger and cannot be updated`,
		},
		{
			name:        "a rename onto the stray financial account",
			account:     model.FinancialAccount{ID: 7, InstitutionID: 10, Name: "Lost and found"},
			newName:     ptr.NewPtr(" untracked accounts"),
			institution: &model.Institution{ID: 10, Name: "Untracked"},
			wantErr:     `"Lost and found" cannot be renamed to " untracked accounts": the account "Untracked accounts" of the institution "Untracked" is managed by the ledger`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			ia := institutionactivity.Institution{}
			input := lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: tc.account.ID, Name: tc.newName, ExternalID: ptr.NewPtr("IBAN")}

			env.OnActivity(aa.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: tc.account.ID}).
				Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: &tc.account}, nil)
			if tc.institution != nil {
				env.OnActivity(ia.Retrieve, mock.Anything, it.RetrieveInstitutionParams{ID: tc.account.InstitutionID}).
					Return(&it.RetrieveInstitutionResult{Institution: tc.institution}, nil).Once()
			}
			if tc.updates {
				env.OnActivity(aa.UpdateFinancialAccountMetadata, mock.Anything, input).
					Return(&lt.FinancialAccountMutationResult{FinancialAccountID: tc.account.ID}, nil).Once()
			}

			env.ExecuteWorkflow(UpdateFinancialAccountMetadata, input)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}

func (s *WorkflowUnitTestSuite) TestUpdateFinancialAccountMetadataPropagatesRetrieveError() {
	input := lt.UpdateFinancialAccountMetadataInput{FinancialAccountID: 7, Name: ptr.NewPtr("Revolut euro")}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: 7}).Return(nil, assert.AnError)

	env.ExecuteWorkflow(UpdateFinancialAccountMetadata, input)

	require.ErrorContains(s.T(), env.GetWorkflowError(), "update financial account metadata")
}
