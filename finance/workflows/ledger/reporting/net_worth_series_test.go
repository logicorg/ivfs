package reporting

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestNetWorthSeries() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	input := lt.NetWorthSeriesParams{From: january, To: january.AddDate(0, 0, 30)}
	series := &lt.NetWorthSeriesResult{
		Balances: []lt.NetWorthBalance{{Date: january, CurrencyID: 1, Assets: decimal.NewFromInt(1000), Liabilities: decimal.NewFromInt(50)}},
		Units:    []lt.NetWorthUnits{{Date: january.AddDate(0, 0, 9), AssetID: 1, Units: decimal.NewFromInt(3)}},
	}

	tests := map[string]struct {
		result  *lt.NetWorthSeriesResult
		err     error
		want    *lt.NetWorthSeriesResult
		wantErr bool
	}{
		"the activity's series": {result: series, want: series},
		"the activity failing":  {err: assert.AnError, wantErr: true},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			ra := activity.Activities{}
			env.OnActivity(ra.NetWorthSeries, mock.Anything, input).Return(tc.result, tc.err)
			env.ExecuteWorkflow(NetWorthSeries, input)
			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var got *lt.NetWorthSeriesResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			assert.Equal(s.T(), tc.want, got)
		})
	}
}
