package accounts

import (
	"testing"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
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

	env.OnActivity(aa.RetrieveFinancialAccount, mock.Anything, act.RetrieveFinancialAccountParams{ID: account.ID}).Return(&act.RetrieveFinancialAccountResult{FinancialAccount: account}, nil)

	env.ExecuteWorkflow(RetrieveFinancialAccount, act.RetrieveFinancialAccountParams{ID: account.ID})

	require.NoError(s.T(), env.GetWorkflowError())
	var retrieved *act.RetrieveFinancialAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), account, retrieved.FinancialAccount)

	env = s.NewTestWorkflowEnvironment()
	env.OnActivity(aa.ListFinancialAccounts, mock.Anything, act.ListFinancialAccountsParams{}).Return(&act.ListFinancialAccountsResult{FinancialAccounts: []model.FinancialAccount{*account}}, nil)
	env.ExecuteWorkflow(ListFinancialAccounts, act.ListFinancialAccountsParams{})
	require.NoError(s.T(), env.GetWorkflowError())
	var listed *act.ListFinancialAccountsResult
	require.NoError(s.T(), env.GetWorkflowResult(&listed))
	require.Len(s.T(), listed.FinancialAccounts, 1)
}

func (s *WorkflowUnitTestSuite) TestCreateLedgerAccount() {
	input := act.CreateLedgerAccountInput{
		FinancialAccountID: ptr.NewPtr(int64(1)),
		CurrencyID:         2,
		Name:               "cash",
		Kind:               "asset",
	}

	env := s.NewTestWorkflowEnvironment()
	aa := activity.Activities{}

	env.OnActivity(aa.CreateLedgerAccount, mock.Anything, input).Return(&act.CreateLedgerAccountResult{LedgerAccountID: 3}, nil)

	env.ExecuteWorkflow(CreateLedgerAccount, input)

	require.NoError(s.T(), env.GetWorkflowError())
	var result *act.CreateLedgerAccountResult
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

	env.OnActivity(aa.RetrieveLedgerAccount, mock.Anything, act.RetrieveLedgerAccountParams{ID: account.ID}).Return(&act.RetrieveLedgerAccountResult{LedgerAccount: account}, nil)

	env.ExecuteWorkflow(RetrieveLedgerAccount, act.RetrieveLedgerAccountParams{ID: account.ID})

	require.NoError(s.T(), env.GetWorkflowError())
	var retrieved *act.RetrieveLedgerAccountResult
	require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
	assert.Equal(s.T(), account, retrieved.LedgerAccount)

	env = s.NewTestWorkflowEnvironment()
	env.OnActivity(aa.ListLedgerAccounts, mock.Anything, act.ListLedgerAccountsParams{}).Return(&act.ListLedgerAccountsResult{LedgerAccounts: []model.LedgerAccount{*account}}, nil)
	env.ExecuteWorkflow(ListLedgerAccounts, act.ListLedgerAccountsParams{})
	require.NoError(s.T(), env.GetWorkflowError())
	var listed *act.ListLedgerAccountsResult
	require.NoError(s.T(), env.GetWorkflowResult(&listed))
	require.Len(s.T(), listed.LedgerAccounts, 1)
}
