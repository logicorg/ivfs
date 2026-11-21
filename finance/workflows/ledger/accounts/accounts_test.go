package accounts

import (
	"testing"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

func (s *WorkflowUnitTestSuite) TestFinancialAccountLifecycleWorkflows() {
	account := &model.FinancialAccount{
		ID:            7,
		InstitutionID: 1,
		Name:          "Revolut EUR",
	}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.RetrieveFinancialAccount, mock.Anything, lt.RetrieveFinancialAccountParams{ID: account.ID}).Return(&lt.RetrieveFinancialAccountResult{FinancialAccount: account}, nil)

	env.ExecuteWorkflow(RetrieveFinancialAccount, lt.RetrieveFinancialAccountParams{ID: account.ID})

	require.NoError(s.T(), env.GetWorkflowError())
	var retrieved *lt.RetrieveFinancialAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), account, retrieved.FinancialAccount)

	env = s.NewTestWorkflowEnvironment()
	env.OnActivity(aa.ListFinancialAccounts, mock.Anything, lt.ListFinancialAccountsParams{}).Return(&lt.ListFinancialAccountsResult{FinancialAccounts: []model.FinancialAccount{*account}}, nil)
	env.ExecuteWorkflow(ListFinancialAccounts, lt.ListFinancialAccountsParams{})
	require.NoError(s.T(), env.GetWorkflowError())
	var listed *lt.ListFinancialAccountsResult
	require.NoError(s.T(), env.GetWorkflowResult(&listed))
	require.Len(s.T(), listed.FinancialAccounts, 1)
}

func (s *WorkflowUnitTestSuite) TestCreateLedgerAccount() {
	input := lt.CreateLedgerAccountInput{
		FinancialAccountID: ptr.NewPtr(int64(1)),
		CurrencyID:         2,
		Name:               "cash",
		Kind:               "asset",
	}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.CreateLedgerAccount, mock.Anything, input).Return(&lt.CreateLedgerAccountResult{LedgerAccountID: 3}, nil)

	env.ExecuteWorkflow(CreateLedgerAccount, input)

	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.CreateLedgerAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	require.Equal(s.T(), int64(3), result.LedgerAccountID)
}

func (s *WorkflowUnitTestSuite) TestLedgerAccountLifecycleWorkflows() {
	account := &model.LedgerAccount{
		ID:         7,
		Name:       "Groceries",
		CurrencyID: 2,
		Kind:       model.LedgerAccountExpense,
	}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, lt.RetrieveLedgerAccountParams{ID: account.ID}).Return(&lt.RetrieveLedgerAccountResult{LedgerAccount: account}, nil)

	env.ExecuteWorkflow(RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: account.ID})

	require.NoError(s.T(), env.GetWorkflowError())
	var retrieved *lt.RetrieveLedgerAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), account, retrieved.LedgerAccount)

	env = s.NewTestWorkflowEnvironment()
	env.OnActivity(aa.ListLedgerAccounts, mock.Anything, lt.ListLedgerAccountsParams{}).Return(&lt.ListLedgerAccountsResult{LedgerAccounts: []model.LedgerAccount{*account}}, nil)
	env.ExecuteWorkflow(ListLedgerAccounts, lt.ListLedgerAccountsParams{})
	require.NoError(s.T(), env.GetWorkflowError())
	var listed *lt.ListLedgerAccountsResult
	require.NoError(s.T(), env.GetWorkflowResult(&listed))
	require.Len(s.T(), listed.LedgerAccounts, 1)
}
