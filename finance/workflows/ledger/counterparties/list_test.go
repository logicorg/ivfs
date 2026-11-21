package counterparties

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/counterparties"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestListCounterparties() {
	counterparty := model.Counterparty{ID: 4, Name: "Tesco"}

	tests := []struct {
		name    string
		result  *lt.ListCounterpartiesResult
		err     error
		want    []model.Counterparty
		wantErr string
	}{
		{name: "success", result: &lt.ListCounterpartiesResult{Counterparties: []model.Counterparty{counterparty}}, want: []model.Counterparty{counterparty}},
		{name: "activity error", err: assert.AnError, wantErr: assert.AnError.Error()},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			env.OnActivity(aa.ListCounterparties, mock.Anything, lt.ListCounterpartiesParams{}).Return(tt.result, tt.err)

			env.ExecuteWorkflow(ListCounterparties, lt.ListCounterpartiesParams{})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tt.wantErr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var listed *lt.ListCounterpartiesResult
			require.NoError(s.T(), env.GetWorkflowResult(&listed))
			assert.Equal(s.T(), tt.want, listed.Counterparties)
		})
	}
}
