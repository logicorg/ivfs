package institution

import (
	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestInsertInstitution() {
	tests := []struct {
		name    string
		params  it.InsertInstitutionParams
		inserts bool
		// insertErr is the error returned by the Insert activity, if any.
		insertErr error
		wantErr   string
	}{
		{
			name:    "an ordinary institution",
			params:  it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"},
			inserts: true,
		},
		{
			name:      "the activity fails",
			params:    it.InsertInstitutionParams{Name: "Revolut", Kind: "bank"},
			inserts:   true,
			insertErr: assert.AnError,
			wantErr:   "insert institution",
		},
		{
			name:    "the untracked institution",
			params:  it.InsertInstitutionParams{Name: " untracked ", Kind: "bank"},
			wantErr: `" untracked " cannot be created: the institution "Untracked" is managed by the ledger; pick another name`,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := institutionactivity.Institution{}

			if tc.inserts {
				var result *it.InsertInstitutionResult
				if tc.insertErr == nil {
					result = &it.InsertInstitutionResult{ID: 1}
				}
				env.OnActivity(aa.Insert, mock.Anything, tc.params).Return(result, tc.insertErr).Once()
			}

			env.ExecuteWorkflow(InsertInstitution, tc.params)

			if tc.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *it.InsertInstitutionResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), int64(1), result.ID)
			env.AssertExpectations(s.T())
		})
	}
}
