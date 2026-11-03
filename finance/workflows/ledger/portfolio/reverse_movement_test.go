package portfolio

import (
	"github.com/google/uuid"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/portfolio"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestReversePositionMovement() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.ReversePositionMovementResult{PositionMovementIDs: []int64{30, 31}}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.ReversePositionMovementInput{MovementID: 10, IdempotencyKey: "reversal-1"}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			pa := activity.Activities{}

			env.OnActivity(pa.ReversePositionMovement, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(ReversePositionMovement, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.ReversePositionMovementResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}

// TestReversePositionMovementDefaultsIdempotencyKey checks that a caller
// leaving IdempotencyKey empty gets a generated one filled in before the
// activity is called, and that two independent executions produce two
// different generated keys.
func (s *WorkflowUnitTestSuite) TestReversePositionMovementDefaultsIdempotencyKey() {
	input := lt.ReversePositionMovementInput{MovementID: 10}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		pa := activity.Activities{}

		env.OnActivity(pa.ReversePositionMovement, mock.Anything, mock.MatchedBy(func(got lt.ReversePositionMovementInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&lt.ReversePositionMovementResult{PositionMovementIDs: []int64{30, 31}}, nil)

		env.ExecuteWorkflow(ReversePositionMovement, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
