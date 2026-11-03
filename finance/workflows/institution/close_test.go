package institution

import (
	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCloseInstitution() {
	tests := []struct {
		name        string
		institution model.Institution
		closes      bool
		// closeErr is the error returned by the Close activity, if any.
		closeErr error
		wantErr  string
	}{
		{
			name:        "an ordinary institution",
			institution: model.Institution{ID: 7, Name: "Revolut", Kind: "bank"},
			closes:      true,
		},
		{
			name:        "the activity fails",
			institution: model.Institution{ID: 7, Name: "Revolut", Kind: "bank"},
			closes:      true,
			closeErr:    assert.AnError,
			wantErr:     "close institution",
		},
		{
			name:        "the untracked institution",
			institution: model.Institution{ID: 7, Name: "Untracked", Kind: "bank"},
			wantErr:     `"Untracked" is managed by the ledger and cannot be closed`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := institutionactivity.Institution{}
			params := it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: tc.institution.ID}}

			env.OnActivity(aa.Retrieve, mock.Anything, params.FindBy).
				Return(&it.RetrieveInstitutionResult{Institution: &tc.institution}, nil)
			if tc.closes {
				env.OnActivity(aa.Close, mock.Anything, params).Return(tc.closeErr).Once()
			}

			env.ExecuteWorkflow(CloseInstitution, params)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}
