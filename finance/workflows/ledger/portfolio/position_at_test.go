package portfolio

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestPositionAt() {
	input := lt.PositionAtParams{FinancialAccountID: 1, AssetID: 2}
	expected := &lt.PositionAtResult{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(3)}
	env := s.NewTestWorkflowEnvironment()
	pa := activity.Activities{}
	env.OnActivity(pa.PositionAt, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(PositionAt, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.PositionAtResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}
