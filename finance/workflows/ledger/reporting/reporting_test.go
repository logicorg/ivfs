package reporting

import (
	"testing"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
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
	input := jt.LedgerAccountBalanceParams{LedgerAccountID: 1}
	expected := &jt.LedgerAccountBalanceResult{LedgerAccountID: 1, CurrencyID: 2, Balance: decimal.NewFromInt(50)}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.LedgerAccountBalance, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(LedgerAccountBalance, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *jt.LedgerAccountBalanceResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestFinancialAccountBalances() {
	input := rpt.FinancialAccountBalancesParams{FinancialAccountID: 1}
	expected := &rpt.FinancialAccountBalancesResult{FinancialAccountID: 1, Balances: []jt.LedgerAccountBalanceResult{{LedgerAccountID: 2, Balance: decimal.NewFromInt(10)}}}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.FinancialAccountBalances, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(FinancialAccountBalances, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *rpt.FinancialAccountBalancesResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestListPositionsAt() {
	input := rpt.PositionsAtParams{FinancialAccountID: 1, AssetID: 2}
	expected := &rpt.PositionsAtResult{Positions: []rpt.PositionResult{{
		FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(4),
		CostBasis: []rpt.CostBasisAmount{{CurrencyID: 3, Amount: decimal.NewFromInt(400)}}, UnknownCostBasis: true,
	}}}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.PositionsAt, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(ListPositionsAt, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *rpt.PositionsAtResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestListPostings() {
	input := jt.ListPostingsParams{LedgerAccountID: 1}
	expected := &jt.ListPostingsResult{}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.ListPostings, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(ListPostings, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *jt.ListPostingsResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestTrialBalanceAtActivityFailure() {
	input := jt.TrialBalanceAtParams{}
	env := s.NewTestWorkflowEnvironment()
	ra := activity.Activities{}
	env.OnActivity(ra.TrialBalanceAt, mock.Anything, input).Return(nil, assert.AnError)
	env.ExecuteWorkflow(TrialBalanceAt, input)
	require.Error(s.T(), env.GetWorkflowError())
}
