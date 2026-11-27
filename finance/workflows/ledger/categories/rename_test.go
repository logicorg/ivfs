package categories

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/categories"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *WorkflowUnitTestSuite) TestRenameCategory() {
	input := ct.RenameCategoryInput{CategoryID: 7, Name: "Food"}
	want := ct.CategoryResult{CategoryID: 7}

	tests := []struct {
		name string
		// fails makes the activity fail.
		fails   bool
		wantErr string
	}{
		{name: "the category is renamed"},
		{name: "the activity fails", fails: true, wantErr: "rename category: activity error"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := activity.Activities{}
			call := env.OnActivity(ca.RenameCategory, mock.Anything, input).Once()
			if tt.fails {
				call.Return(nil, temporal.NewNonRetryableApplicationError("boom", "ErrInvalidParameter", nil))
			} else {
				call.Return(&want, nil)
			}

			env.ExecuteWorkflow(RenameCategory, input)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result ct.CategoryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), want, result)
			env.AssertExpectations(s.T())
		})
	}
}
