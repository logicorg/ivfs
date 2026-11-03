package reporting

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestCategoryTotals() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	input := lt.CategoryTotalsParams{From: january, To: january.AddDate(0, 1, 0)}
	totals := &lt.CategoryTotalsResult{Totals: []lt.CategoryTotal{
		{LedgerAccountID: 1, Kind: model.LedgerAccountExpense, CurrencyID: 2, CurrencyCode: "EUR", Total: decimal.NewFromInt(120)},
	}}

	tests := map[string]struct {
		result  *lt.CategoryTotalsResult
		err     error
		want    *lt.CategoryTotalsResult
		wantErr bool
	}{
		"the activity's totals": {result: totals, want: totals},
		"the activity failing":  {err: assert.AnError, wantErr: true},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			ra := activity.Activities{}
			env.OnActivity(ra.CategoryTotals, mock.Anything, input).Return(tc.result, tc.err)
			env.ExecuteWorkflow(CategoryTotals, input)
			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var got *lt.CategoryTotalsResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			assert.Equal(s.T(), tc.want, got)
		})
	}
}
