package counterparties

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/counterparties"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestRetrieveCounterparty() {
	counterparty := &model.Counterparty{ID: 4, Name: "Tesco"}
	params := lt.RetrieveCounterpartyParams{ID: counterparty.ID}

	tests := []struct {
		name    string
		result  *lt.RetrieveCounterpartyResult
		err     error
		want    *model.Counterparty
		wantErr string
	}{
		{name: "success", result: &lt.RetrieveCounterpartyResult{Counterparty: counterparty}, want: counterparty},
		{name: "activity error", err: assert.AnError, wantErr: assert.AnError.Error()},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			env.OnActivity(aa.RetrieveCounterparty, mock.Anything, params).Return(tt.result, tt.err)

			env.ExecuteWorkflow(RetrieveCounterparty, params)

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var retrieved *lt.RetrieveCounterpartyResult
			require.NoError(s.T(), env.GetWorkflowResult(&retrieved))
			assert.Equal(s.T(), tt.want, retrieved.Counterparty)
		})
	}
}
