package portfolio

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestDeletePositionMovement() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.DeletePositionMovementResult{}, nil},
		},
		{
			name:   "success with warnings",
			result: []any{&lt.DeletePositionMovementResult{Warnings: []lt.NegativeHolding{{FinancialAccountID: 1, FinancialAccountName: "Broker", AssetID: 2, AssetName: "ACME", At: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-1)}}}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.DeletePositionMovementInput{MovementID: 10}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.DeletePositionMovement, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(DeletePositionMovement, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.DeletePositionMovementResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}
