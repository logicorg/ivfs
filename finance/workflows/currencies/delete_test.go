package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestDeleteCurrency() {
	tests := []struct {
		name   string
		params ct.DeleteCurrencyParams
		err    error
	}{
		{name: "by id", params: ct.DeleteCurrencyParams{ID: 3}},
		{name: "by code", params: ct.DeleteCurrencyParams{Code: "EUR"}},
		{name: "activity failure", params: ct.DeleteCurrencyParams{ID: 3}, err: assert.AnError},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := &currency.Currency{}

			env.OnActivity(ca.DeleteCurrency, mock.Anything, tc.params).Return(tc.err)

			env.ExecuteWorkflow(DeleteCurrency, &tc.params)

			if tc.err != nil {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
		})
	}
}
