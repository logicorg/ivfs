package institution

import (
	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestUpdateInstitution() {
	tests := []struct {
		name        string
		institution model.Institution
		newName     string
		newKind     string
		updates     bool
		wantErr     string
	}{
		{
			name:        "a rename",
			institution: model.Institution{ID: 7, Name: "Revolut", Kind: "bank"},
			newName:     "Revolut Ltd",
			updates:     true,
		},
		{
			name:        "a kind change",
			institution: model.Institution{ID: 7, Name: "Revolut", Kind: "bank"},
			newKind:     "broker",
			updates:     true,
		},
		{
			name:        "the untracked institution",
			institution: model.Institution{ID: 7, Name: "Untracked", Kind: "bank"},
			newKind:     "broker",
			wantErr:     `"Untracked" is managed by the ledger and cannot be updated`,
		},
		{
			name:        "a rename onto the untracked institution",
			institution: model.Institution{ID: 7, Name: "Revolut", Kind: "bank"},
			newName:     "UNTRACKED",
			wantErr:     `"Revolut" cannot be renamed to "UNTRACKED": the institution "Untracked" is managed by the ledger`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := institutionactivity.Institution{}
			params := it.UpdateInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: tc.institution.ID}, Name: tc.newName, Kind: tc.newKind}

			env.OnActivity(aa.Retrieve, mock.Anything, params.FindBy).
				Return(&it.RetrieveInstitutionResult{Institution: &tc.institution}, nil)
			if tc.updates {
				env.OnActivity(aa.Update, mock.Anything, params).Return(&it.UpdateInstitutionResult{ID: tc.institution.ID}, nil).Once()
			}

			env.ExecuteWorkflow(UpdateInstitution, params)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}

func (s *WorkflowUnitTestSuite) TestUpdateInstitutionPropagatesRetrieveError() {
	params := it.UpdateInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 7}, Name: "Revolut Ltd"}

	env := s.NewTestWorkflowEnvironment()
	aa := institutionactivity.Institution{}

	env.OnActivity(aa.Retrieve, mock.Anything, params.FindBy).Return(nil, assert.AnError)

	env.ExecuteWorkflow(UpdateInstitution, params)

	require.ErrorContains(s.T(), env.GetWorkflowError(), "update institution")
}
