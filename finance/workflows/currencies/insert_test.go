package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestInsertCurrency() {
	params := ct.InsertCurrencyParams{Code: "EUR", Name: "Euro", Symbol: "€", Precision: ptr.NewPtr(2)}

	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{name: "success", result: []any{&ct.InsertCurrencyResult{CurrencyID: 3}, nil}},
		{name: "activity failure", result: []any{nil, assert.AnError}, wantErr: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := &currency.Currency{}

			env.OnActivity(ca.InsertCurrency, mock.Anything, params).Return(tc.result...)

			env.ExecuteWorkflow(InsertCurrency, params)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result ct.InsertCurrencyResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), ct.InsertCurrencyResult{CurrencyID: 3}, result)
		})
	}
}
