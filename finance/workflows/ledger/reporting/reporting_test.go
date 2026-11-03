package reporting

import (
	"testing"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
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

func (s *WorkflowUnitTestSuite) TestLedgerAccountBalance() {
	input := lt.LedgerAccountBalanceParams{LedgerAccountID: 1}
	expected := &lt.LedgerAccountBalanceResult{LedgerAccountID: 1, CurrencyID: 2, Balance: decimal.NewFromInt(50)}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.LedgerAccountBalance, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(LedgerAccountBalance, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.LedgerAccountBalanceResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestFinancialAccountBalances() {
	input := lt.FinancialAccountBalancesParams{FinancialAccountID: 1}
	expected := &lt.FinancialAccountBalancesResult{FinancialAccountID: 1, Balances: []lt.LedgerAccountBalanceResult{{LedgerAccountID: 2, Balance: decimal.NewFromInt(10)}}}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.FinancialAccountBalances, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(FinancialAccountBalances, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.FinancialAccountBalancesResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestListPositionsAt() {
	input := lt.PositionsAtParams{FinancialAccountID: 1, AssetID: 2}
	expected := &lt.PositionsAtResult{Positions: []lt.PositionResult{{
		FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(4),
		CostBasis: []lt.CostBasisAmount{{CurrencyID: 3, Amount: decimal.NewFromInt(400)}}, UnknownCostBasis: true,
	}}}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.PositionsAt, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(ListPositionsAt, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.PositionsAtResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestListPostings() {
	input := lt.ListPostingsParams{LedgerAccountID: 1}
	expected := &lt.ListPostingsResult{}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.ListPostings, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(ListPostings, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.ListPostingsResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestTrialBalanceAtActivityFailure() {
	input := lt.TrialBalanceAtParams{}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.TrialBalanceAt, mock.Anything, input).Return(nil, assert.AnError)
	env.ExecuteWorkflow(TrialBalanceAt, input)
	require.Error(s.T(), env.GetWorkflowError())
}
