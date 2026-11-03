package categories

import (
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRetrieveCategories() {
	account := &model.LedgerAccount{ID: 10, Name: "Groceries", CurrencyID: 2, Kind: model.LedgerAccountExpense}

	env := s.NewTestWorkflowEnvironment()

	retrieveInput := ct.RetrieveCategoryParams{ID: 10}
	env.OnWorkflow(ledgeraccountswf.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: 10}).
		Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: account}, nil)
	env.ExecuteWorkflow(RetrieveCategory, retrieveInput)

	require.NoError(s.T(), env.GetWorkflowError())

	var retrieved *ct.RetrieveCategoryResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), account, retrieved.LedgerAccount)
}

func (s *WorkflowUnitTestSuite) TestListCategories() {
	account := &model.LedgerAccount{ID: 10, Name: "Groceries", CurrencyID: 2, Kind: model.LedgerAccountExpense}

	env := s.NewTestWorkflowEnvironment()

	listInput := ct.ListCategoriesParams{CurrencyID: ptr.Int64(2), NameContains: "grocer"}
	env.OnWorkflow(ledgeraccountswf.ListLedgerAccounts, mock.Anything, lt.ListLedgerAccountsParams{
		CurrencyID: ptr.Int64(2), NameContains: "grocer",
		Kinds: []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense},
	}).Return(&lt.ListLedgerAccountsResult{LedgerAccounts: []model.LedgerAccount{*account}}, nil)
	env.ExecuteWorkflow(ListCategories, listInput)

	require.NoError(s.T(), env.GetWorkflowError())

	var listed *ct.ListCategoriesResult
	require.NoError(s.T(), env.GetWorkflowResult(&listed))
	require.Len(s.T(), listed.LedgerAccounts, 1)
}
