package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestUpdateCurrency() {
	params := ct.UpdateCurrencyParams{ID: 3, Name: ptr.NewPtr("European Euro"), Symbol: ptr.NewPtr("E")}

	tests := []struct {
		name string
		err  error
	}{
		{name: "success"},
		{name: "activity failure", err: assert.AnError},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := &currency.Currency{}

			env.OnActivity(ca.UpdateCurrency, mock.Anything, params).Return(tc.err)

			env.ExecuteWorkflow(UpdateCurrency, &params)

			if tc.err != nil {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
		})
	}
}
