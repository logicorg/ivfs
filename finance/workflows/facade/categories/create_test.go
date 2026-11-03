package categories

import (
	"errors"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCreateCategory() {
	input := ct.CreateCategoryInput{Name: "Groceries", CurrencyID: 2, Kind: model.LedgerAccountExpense}
	env := s.NewTestWorkflowEnvironment()
	env.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, lt.CreateLedgerAccountInput{
		Name: "Groceries", CurrencyID: 2, Kind: model.LedgerAccountExpense,
	}).Return(&lt.CreateLedgerAccountResult{LedgerAccountID: 10}, nil)
	env.ExecuteWorkflow(CreateCategory, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *ct.CategoryResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), int64(10), result.LedgerAccountID)

	failing := s.NewTestWorkflowEnvironment()
	failing.OnWorkflow(ledgeraccountswf.CreateLedgerAccount, mock.Anything, mock.Anything).
		Return(nil, errors.New("database down"))
	failing.ExecuteWorkflow(CreateCategory, input)
	require.ErrorContains(s.T(), failing.GetWorkflowError(), "create category")

	invalid := s.NewTestWorkflowEnvironment()
	invalid.ExecuteWorkflow(CreateCategory, ct.CreateCategoryInput{Name: "Assets", CurrencyID: 2, Kind: model.LedgerAccountAsset})
	require.Error(s.T(), invalid.GetWorkflowError())
	assert.Contains(s.T(), invalid.GetWorkflowError().Error(), `category kind must be "income" or "expense"`)
}
