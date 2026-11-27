package categories

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/categories"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *WorkflowUnitTestSuite) TestDeleteCategory() {
	input := ct.DeleteCategoryInput{CategoryID: 7, ReplacementID: ptr.Int64(9)}

	tests := []struct {
		name string
		// fails makes the activity fail.
		fails   bool
		wantErr string
	}{
		{name: "the category is deleted"},
		{name: "the activity fails", fails: true, wantErr: "delete category: activity error"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := activity.Activities{}
			call := env.OnActivity(ca.DeleteCategory, mock.Anything, input).Once()
			if tt.fails {
				call.Return(temporal.NewNonRetryableApplicationError("boom", "ErrInvalidParameter", nil))
			} else {
				call.Return(nil)
			}

			env.ExecuteWorkflow(DeleteCategory, input)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			env.AssertExpectations(s.T())
		})
	}
}
