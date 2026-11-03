package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// TestRetrieveCurrency checks that a currency is retrieved by ID when one is
// given, and found by name, code, or symbol otherwise, and that parameters
// that cannot select a currency are rejected before any activity runs.
func (s *WorkflowUnitTestSuite) TestRetrieveCurrency() {
	ca := (*currency.Currency)(nil)
	euro := &model.Currency{ID: 3, Code: "EUR", Name: "Euro", Symbol: "€", Precision: 2}

	tests := []struct {
		name   string
		params ct.RetrieveCurrencyParams
		// activity is the activity the workflow is expected to call with arg,
		// returning result. It is nil when no activity may run.
		activity any
		arg      any
		result   []any
		errStr   string
	}{
		{
			name:     "by id",
			params:   ct.RetrieveCurrencyParams{ID: 3},
			activity: ca.RetrieveCurrencyByID, arg: int64(3), result: []any{euro, nil},
		},
		{
			name:     "an id wins over a code",
			params:   ct.RetrieveCurrencyParams{ID: 3, Code: "USD"},
			activity: ca.RetrieveCurrencyByID, arg: int64(3), result: []any{euro, nil},
		},
		{
			name:     "by code",
			params:   ct.RetrieveCurrencyParams{Code: "EUR"},
			activity: ca.FindCurrencyBy, arg: ct.FindCurrencyByParams{Code: "EUR"}, result: []any{euro, nil},
		},
		{
			name:     "by name and symbol",
			params:   ct.RetrieveCurrencyParams{Name: "Euro", Symbol: "€"},
			activity: ca.FindCurrencyBy, arg: ct.FindCurrencyByParams{Name: "Euro", Symbol: "€"}, result: []any{euro, nil},
		},
		{
			name:   "nothing to select by",
			params: ct.RetrieveCurrencyParams{},
			errStr: "must be supplied",
		},
		{
			name:   "a negative id",
			params: ct.RetrieveCurrencyParams{ID: -1, Code: "EUR"},
			errStr: "id cannot be negative",
		},
		{
			name:     "activity failure by id",
			params:   ct.RetrieveCurrencyParams{ID: 3},
			activity: ca.RetrieveCurrencyByID, arg: int64(3), result: []any{nil, assert.AnError},
			errStr: assert.AnError.Error(),
		},
		{
			name:     "activity failure by code",
			params:   ct.RetrieveCurrencyParams{Code: "EUR"},
			activity: ca.FindCurrencyBy, arg: ct.FindCurrencyByParams{Code: "EUR"}, result: []any{nil, assert.AnError},
			errStr: assert.AnError.Error(),
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.activity != nil {
				env.OnActivity(tc.activity, mock.Anything, tc.arg).Return(tc.result...)
			}

			env.ExecuteWorkflow(RetrieveCurrency, &tc.params)

			if tc.errStr != "" {
				require.ErrorContains(s.T(), env.GetWorkflowError(), tc.errStr)
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *model.Currency
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), euro, result)
		})
	}
}
