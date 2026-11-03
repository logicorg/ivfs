package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestListCurrencies() {
	currencies := []*model.Currency{
		{ID: 1, Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2},
		{ID: 2, Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Precision: 0},
	}

	tests := []struct {
		name    string
		result  []any
		want    *ct.ListCurrenciesResult
		wantErr bool
	}{
		{name: "success", result: []any{currencies, nil}, want: &ct.ListCurrenciesResult{Currencies: currencies}},
		{name: "no currencies", result: []any{[]*model.Currency{}, nil}, want: &ct.ListCurrenciesResult{Currencies: []*model.Currency{}}},
		{name: "activity failure", result: []any{nil, assert.AnError}, wantErr: true},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ca := &currency.Currency{}

			env.OnActivity(ca.ListCurrencies, mock.Anything).Return(tc.result...)

			env.ExecuteWorkflow(ListCurrencies, ct.ListCurrenciesParams{})

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *ct.ListCurrenciesResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), tc.want, result)
		})
	}
}
