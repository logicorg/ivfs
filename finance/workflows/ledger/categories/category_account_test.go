package categories

import (
	"errors"

	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

// categoryAccount runs CategoryAccount as a workflow would.
func categoryAccount(ctx workflow.Context, categoryID int64, kind model.LedgerAccountKind, currencyID int64) (int64, error) {
	return CategoryAccount(ctx, categoryID, kind, currencyID)
}

func (s *WorkflowUnitTestSuite) TestCategoryAccount() {
	groceries := model.Category{ID: 20, Name: "Groceries", Kind: model.LedgerAccountExpense}
	salary := model.Category{ID: 30, Name: "Salary", Kind: model.LedgerAccountIncome}

	tests := []struct {
		name     string
		category model.Category
		kind     model.LedgerAccountKind
		// wantAccount is the name of the account looked for, none when the
		// category is refused first.
		wantAccount string
		retrieveErr string
		createErr   string
		wantError   string
	}{
		{name: "an expense", category: groceries, kind: model.LedgerAccountExpense, wantAccount: "Expenses"},
		{name: "an income", category: salary, kind: model.LedgerAccountIncome, wantAccount: "Income"},
		{
			name: "a category of the other kind", category: salary, kind: model.LedgerAccountExpense,
			wantError: `"Salary" is an income category, not an expense one`,
		},
		{
			name: "an unknown category", category: groceries, kind: model.LedgerAccountExpense,
			retrieveErr: "resource Category not found with ID: 20", wantError: "retrieve category 20: ",
		},
		{
			name: "an unknown currency", category: groceries, kind: model.LedgerAccountExpense, wantAccount: "Expenses",
			createErr: "resource Currency not found with ID: 2", wantError: "find the Expenses account of currency 2: ",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(categoryAccount)
			retrieval := env.OnWorkflow(RetrieveCategory, mock.Anything, ct.RetrieveCategoryParams{ID: tc.category.ID}).Once()
			if tc.retrieveErr != "" {
				retrieval.Return(nil, errors.New(tc.retrieveErr))
			} else {
				retrieval.Return(&ct.RetrieveCategoryResult{Category: &tc.category}, nil)
			}
			if tc.wantAccount != "" {
				creation := env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything,
					act.CreateLedgerAccountInput{CurrencyID: 2, Name: tc.wantAccount, Kind: tc.kind}).Once()
				if tc.createErr != "" {
					creation.Return(nil, errors.New(tc.createErr))
				} else {
					creation.Return(&act.CreateLedgerAccountResult{LedgerAccountID: 200}, nil)
				}
			}

			env.ExecuteWorkflow(categoryAccount, tc.category.ID, tc.kind, int64(2))

			if tc.wantError != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
				env.AssertExpectations(s.T())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var got int64
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), int64(200), got)
			env.AssertExpectations(s.T())
		})
	}
}
