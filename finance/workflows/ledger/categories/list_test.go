package categories

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/categories"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/temporal"
)

func (s *WorkflowUnitTestSuite) TestListCategories() {
	input := ct.ListCategoriesParams{Kind: model.LedgerAccountExpense, NameContains: "gro", IncludeClosed: true}
	want := ct.ListCategoriesResult{Categories: []model.Category{{ID: 7, Name: "Groceries", Kind: model.LedgerAccountExpense}}}

	tests := []struct {
		name string
		// fails makes the activity fail.
		fails   bool
		wantErr string
	}{
		{name: "the categories are returned"},
		{name: "the activity fails", fails: true, wantErr: "list categories: activity error"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := activity.Activities{}
			call := env.OnActivity(ca.ListCategories, mock.Anything, input).Once()
			if tt.fails {
				call.Return(nil, temporal.NewNonRetryableApplicationError("boom", "ErrInvalidParameter", nil))
			} else {
				call.Return(&want, nil)
			}

			env.ExecuteWorkflow(ListCategories, input)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result ct.ListCategoriesResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), want, result)
			env.AssertExpectations(s.T())
		})
	}
}
