package manual

import (
	"testing"
	"time"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows/assets/assetvalues"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValues(t *testing.T) {
	asset := model.Asset{ID: 7, Name: "Villa"}
	observed := func(id int64, at time.Time, currencyID, amount int64) *model.AssetValue {
		return &model.AssetValue{ID: id, AssetID: 7, CurrencyID: currencyID, Value: decimal.NewFromInt(amount), ObservedAt: at}
	}
	lastWeek := observed(1, time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC), 1, 100)
	saturday := observed(2, time.Date(2026, 10, 10, 15, 0, 0, 0, time.UTC), 2, 101)
	lateMonday := observed(3, time.Date(2026, 10, 12, 23, 30, 0, 0, time.UTC), 1, 102)
	value := func(date time.Time, v *model.AssetValue, currency string) assetvalue.Value {
		return assetvalue.Value{Date: date, Amount: v.Value, Currency: currency, ObservedAt: v.ObservedAt}
	}
	// history is the workflow reading the values from the first date to the
	// end of the last one.
	history := func(from, to time.Time, values ...*model.AssetValue) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: assetvalues.RetrieveAssetValueHistory,
			Params: avt.RetrieveAssetValueHistoryParams{
				AssetID: 7, From: from, To: to.AddDate(0, 0, 1).Add(-time.Nanosecond),
			},
			Result: avt.RetrieveAssetValueHistoryResult{AssetValues: values},
		}
	}
	fridayToMonday := assetvalue.Range{From: date(2026, 10, 9), To: date(2026, 10, 12)}

	tests := map[string]struct {
		r            assetvalue.Range
		calls        []*apitest.WorkflowCall
		want         []assetvalue.Value
		wantNotFound string
		wantErr      string
	}{
		"the value carried in, then each day's latest": {
			r:     fridayToMonday,
			calls: []*apitest.WorkflowCall{history(date(2026, 10, 9), date(2026, 10, 12), lastWeek, saturday, lateMonday), listCurrencies()},
			want: []assetvalue.Value{
				value(date(2026, 10, 9), lastWeek, "EUR"),
				value(date(2026, 10, 10), saturday, "USD"),
				value(date(2026, 10, 11), saturday, "USD"),
				value(date(2026, 10, 12), lateMonday, "EUR"),
			},
		},
		"values that start within the range": {
			r:     fridayToMonday,
			calls: []*apitest.WorkflowCall{history(date(2026, 10, 9), date(2026, 10, 12), saturday), listCurrencies()},
			want: []assetvalue.Value{
				value(date(2026, 10, 10), saturday, "USD"),
				value(date(2026, 10, 11), saturday, "USD"),
				value(date(2026, 10, 12), saturday, "USD"),
			},
		},
		"every week, from its first Sunday": {
			r:     assetvalue.Range{From: date(2026, 10, 7), To: date(2026, 10, 21), Interval: assetvalue.Weekly},
			calls: []*apitest.WorkflowCall{history(date(2026, 10, 11), date(2026, 10, 21), lastWeek, saturday, lateMonday), listCurrencies()},
			want: []assetvalue.Value{
				value(date(2026, 10, 11), saturday, "USD"),
				value(date(2026, 10, 18), lateMonday, "EUR"),
				value(date(2026, 10, 21), lateMonday, "EUR"),
			},
		},
		"no values, without listing the currencies": {
			r:            fridayToMonday,
			calls:        []*apitest.WorkflowCall{history(date(2026, 10, 9), date(2026, 10, 12))},
			wantNotFound: "AssetValue",
		},
		"a value in a currency the user does not have": {
			r:            fridayToMonday,
			calls:        []*apitest.WorkflowCall{history(date(2026, 10, 9), date(2026, 10, 12), observed(4, saturday.ObservedAt, 9, 1)), listCurrencies()},
			wantNotFound: "Currency",
		},
		"a history that fails": {
			r: fridayToMonday,
			calls: []*apitest.WorkflowCall{func() *apitest.WorkflowCall {
				call := history(date(2026, 10, 9), date(2026, 10, 12))
				call.Result, call.Err = nil, assert.AnError
				return call
			}()},
			wantErr: "the workflow failed",
		},
		"a range that ends before it starts, without running a workflow": {
			r:       assetvalue.Range{From: date(2026, 10, 12), To: date(2026, 10, 9)},
			wantErr: "the range ends on 2026-10-09, before it starts on 2026-10-12",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p, ctx := provider(t, tc.calls...)

			got, err := p.Values(ctx, asset, tc.r)

			switch {
			case tc.wantNotFound != "":
				var notFound errors.ErrNotFound
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, tc.wantNotFound, notFound.ResourceType)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
