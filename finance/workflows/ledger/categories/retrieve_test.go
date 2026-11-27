package categories

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/categories"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *WorkflowUnitTestSuite) TestRetrieveCategory() {
	input := ct.RetrieveCategoryParams{ID: 7}
	want := ct.RetrieveCategoryResult{Category: &model.Category{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense, ParentID: ptr.Int64(3)}}

	tests := []struct {
		name string
		// fails makes the activity fail.
		fails   bool
		wantErr string
	}{
		{name: "the category is returned"},
		{name: "the activity fails", fails: true, wantErr: "retrieve category: activity error"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := activity.Activities{}
			call := env.OnActivity(ca.RetrieveCategory, mock.Anything, input).Once()
			if tt.fails {
				call.Return(nil, temporal.NewNonRetryableApplicationError("boom", "ErrInvalidParameter", nil))
			} else {
				call.Return(&want, nil)
			}

			env.ExecuteWorkflow(RetrieveCategory, input)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result ct.RetrieveCategoryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), want, result)
			env.AssertExpectations(s.T())
		})
	}
}
