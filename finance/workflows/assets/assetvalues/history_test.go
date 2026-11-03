package assetvalues

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type HistoryWorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestHistoryWorkflowUnitTestSuite(t *testing.T) {
	suite.Run(t, new(HistoryWorkflowUnitTestSuite))
}

func (s *HistoryWorkflowUnitTestSuite) TestRetrieveAssetValueHistory() {
	from := time.Date(2026, 10, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 10, 11, 23, 59, 59, 0, time.UTC)
	observed := func(id int64, at time.Time, amount int64) *model.AssetValue {
		return &model.AssetValue{ID: id, AssetID: 7, CurrencyID: 1, Value: decimal.NewFromInt(amount), ObservedAt: at}
	}
	before := observed(1, from.Add(-72*time.Hour), 100)
	atFrom := observed(2, from, 101)
	within := observed(3, from.Add(48*time.Hour), 102)
	lastSecond := observed(4, to, 103)
	after := observed(5, to.Add(time.Second), 104)

	retrieveAt := avt.RetrieveAssetValueParams{AssetID: 7, Date: &from}
	retrieveSince := avt.RetrieveAssetValuesParams{AssetID: 7, Since: &from}
	notFound := errors.ErrNotFound{ResourceType: "AssetValue"}
	week := avt.RetrieveAssetValueHistoryParams{AssetID: 7, From: from, To: to}

	tests := map[string]struct {
		params avt.RetrieveAssetValueHistoryParams
		// carriedIn and since are what the activities answer; a nil
		// carriedIn with no error means the workflow must not call them.
		carriedIn    *model.AssetValue
		carriedInErr error
		since        []*model.AssetValue
		sinceErr     error
		want         []*model.AssetValue
		wantErr      string
	}{
		"the value carried in, then those in the range, up to its last second": {
			params: week, carriedIn: before, since: []*model.AssetValue{within, lastSecond, after},
			want: []*model.AssetValue{before, within, lastSecond},
		},
		"nothing before the range": {
			params: week, carriedInErr: notFound, since: []*model.AssetValue{within},
			want: []*model.AssetValue{within},
		},
		"a value observed as the range starts, once": {
			params: week, carriedIn: atFrom, since: []*model.AssetValue{atFrom, within},
			want: []*model.AssetValue{atFrom, within},
		},
		"no values at all": {
			params: week, carriedInErr: notFound, since: []*model.AssetValue{},
			want: []*model.AssetValue{},
		},
		"only values after the range": {
			params: week, carriedInErr: notFound, since: []*model.AssetValue{after},
			want: []*model.AssetValue{},
		},
		"the value at the start fails": {
			params: week, carriedInErr: errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "AssetID"}},
			since:   []*model.AssetValue{within},
			wantErr: "retrieve the asset value at 2026-10-05T00:00:00Z",
		},
		"the values since the start fail": {
			params: week, carriedIn: before, sinceErr: errors.ErrMissingRequiredParameter{Parameters: []string{"AssetID"}},
			wantErr: "retrieve the asset values since 2026-10-05T00:00:00Z",
		},
		"no asset, without calling the activities": {
			params:  avt.RetrieveAssetValueHistoryParams{From: from, To: to},
			wantErr: "missing required parameters: AssetID",
		},
		"a range that ends before it starts, without calling the activities": {
			params:  avt.RetrieveAssetValueHistoryParams{AssetID: 7, From: to, To: from},
			wantErr: "the history ends at 2026-10-05T00:00:00Z, before it starts at 2026-10-11T23:59:59Z",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			var ava *value.AssetValueActivity
			if tc.carriedIn != nil || tc.carriedInErr != nil {
				var result *avt.RetrieveAssetValueResult
				if tc.carriedIn != nil {
					result = &avt.RetrieveAssetValueResult{AssetValue: tc.carriedIn}
				}
				env.OnActivity(ava.RetrieveAssetValue, mock.Anything, retrieveAt).Return(result, tc.carriedInErr)
				var values *avt.RetrieveAssetValuesResult
				if tc.since != nil {
					values = &avt.RetrieveAssetValuesResult{AssetValues: tc.since}
				}
				env.OnActivity(ava.RetrieveAssetValues, mock.Anything, retrieveSince).Return(values, tc.sinceErr)
			}

			env.ExecuteWorkflow(RetrieveAssetValueHistory, tc.params)

			err := env.GetWorkflowError()
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				env.AssertExpectations(s.T())
				return
			}
			require.NoError(s.T(), err)
			var result *avt.RetrieveAssetValueHistoryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Equal(s.T(), &avt.RetrieveAssetValueHistoryResult{AssetValues: tc.want}, result)
		})
	}
}
