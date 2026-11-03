package portfolio

import (
	"time"

	"github.com/google/uuid"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRegisterPositionMovement() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.RegisterPositionMovementResult{PositionMovementIDs: []int64{10, 11}}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.RegisterPositionMovementInput{
		IdempotencyKey: "transfer-1",
		Kind:           model.PositionMovementTransfer,
		AssetID:        2,
		OccurredAt:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Legs: []lt.PositionMovementLeg{
			{FinancialAccountID: 1, UnitsDelta: decimal.NewFromInt(-1)},
			{FinancialAccountID: 3, UnitsDelta: decimal.NewFromInt(1)},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.RegisterPositionMovement, mock.Anything, mock.MatchedBy(func(got lt.RegisterPositionMovementInput) bool {
				return got.IdempotencyKey == input.IdempotencyKey && len(got.Legs) == len(input.Legs)
			})).Return(tc.result...)

			env.ExecuteWorkflow(RegisterPositionMovement, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.RegisterPositionMovementResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestRegisterPositionMovementDefaultsIdempotencyKey checks that a caller
// leaving IdempotencyKey empty gets a generated one filled in before the
// activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestRegisterPositionMovementDefaultsIdempotencyKey() {
	input := lt.RegisterPositionMovementInput{
		Kind:       model.PositionMovementTransfer,
		AssetID:    2,
		OccurredAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Legs: []lt.PositionMovementLeg{
			{FinancialAccountID: 1, UnitsDelta: decimal.NewFromInt(-1)},
			{FinancialAccountID: 3, UnitsDelta: decimal.NewFromInt(1)},
		},
	}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		pa := activity.Activities{}

		env.OnActivity(pa.RegisterPositionMovement, mock.Anything, mock.MatchedBy(func(got lt.RegisterPositionMovementInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&lt.RegisterPositionMovementResult{PositionMovementIDs: []int64{1, 2}}, nil)

		env.ExecuteWorkflow(RegisterPositionMovement, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
