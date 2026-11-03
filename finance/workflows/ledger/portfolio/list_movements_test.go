package portfolio

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestListPositionMovements() {
	page := &lt.ListPositionMovementsResult{
		Movements: []model.PositionMovement{{ID: 10, FinancialAccountID: 1, AssetID: 2, UnitsDelta: decimal.NewFromInt(3)}},
		Next:      &lt.PositionMovementCursor{OccurredAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), ID: 10},
	}

	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{name: "success", result: []any{page, nil}},
		{name: "activity failure", result: []any{nil, assert.AnError}, wantErr: true},
	}

	params := lt.ListPositionMovementsParams{FinancialAccountID: 1, AssetID: 2, Limit: 1}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.ListPositionMovements, mock.Anything, params).Return(tc.result...)

			env.ExecuteWorkflow(ListPositionMovements, params)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.ListPositionMovementsResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Len(s.T(), result.Movements, 1)
			assert.Equal(s.T(), page.Movements[0].ID, result.Movements[0].ID)
			assert.Equal(s.T(), page.Next, result.Next)
		})
	}
}
