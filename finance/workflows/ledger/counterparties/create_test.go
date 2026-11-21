package counterparties

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/counterparties"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCreateCounterparty() {
	input := lt.CreateCounterpartyInput{Name: "Tesco"}

	tests := []struct {
		name    string
		result  *lt.CreateCounterpartyResult
		err     error
		wantID  int64
		wantErr string
	}{
		{name: "success", result: &lt.CreateCounterpartyResult{CounterpartyID: 4}, wantID: 4},
		{name: "activity error", err: assert.AnError, wantErr: assert.AnError.Error()},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			env.OnActivity(aa.CreateCounterparty, mock.Anything, input).Return(tt.result, tt.err)

			env.ExecuteWorkflow(CreateCounterparty, input)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.CreateCounterpartyResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), tt.wantID, result.CounterpartyID)
		})
	}
}
