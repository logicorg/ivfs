package manual

import (
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/conversiontypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRates(t *testing.T) {
	fixing := func(id int64, at time.Time, rate string) *model.Conversion {
		return &model.Conversion{ID: id, FromCurrencyID: 1, ToCurrencyID: 2, Rate: decimal.RequireFromString(rate), ObservedAt: at}
	}
	thursday := fixing(1, time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC), "1.0911")
	friday := fixing(2, time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC), "1.0921")
	monday := fixing(3, time.Date(2026, 10, 12, 16, 0, 0, 0, time.UTC), "1.0931")
	rate := func(date time.Time, c *model.Conversion) assetvalue.Rate {
		return assetvalue.Rate{Date: date, Rate: c.Rate, ObservedAt: c.ObservedAt}
	}
	fridayToMonday := assetvalue.Range{From: date(2026, 10, 9), To: date(2026, 10, 12)}
	history := func(rates ...*model.Conversion) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: currencieswf.RetrieveConversionRateHistory,
			Params:   ct.RetrieveRateHistoryParams{FromCurrency: 1, ToCurrency: 2, From: date(2026, 10, 9), To: date(2026, 10, 12)},
			Result:   ct.RetrieveRateHistoryResult{Conversions: rates},
		}
	}

	tests := map[string]struct {
		base, quote  string
		r            assetvalue.Range
		calls        []*apitest.WorkflowCall
		want         []assetvalue.Rate
		wantNotFound bool
		wantErr      string
	}{
		"the rate carried in, then carried over the weekend": {
			base: "EUR", quote: "USD", r: fridayToMonday,
			calls: []*apitest.WorkflowCall{listCurrencies(), history(thursday, friday, monday)},
			want: []assetvalue.Rate{
				rate(date(2026, 10, 9), friday),
				rate(date(2026, 10, 10), friday),
				rate(date(2026, 10, 11), friday),
				rate(date(2026, 10, 12), monday),
			},
		},
		"rates that start within the range": {
			base: "EUR", quote: "USD", r: fridayToMonday,
			calls: []*apitest.WorkflowCall{listCurrencies(), history(monday)},
			want:  []assetvalue.Rate{rate(date(2026, 10, 12), monday)},
		},
		"no rates": {
			base: "EUR", quote: "USD", r: fridayToMonday,
			calls:        []*apitest.WorkflowCall{listCurrencies(), history()},
			wantNotFound: true,
		},
		"a currency the user does not have, without reading rates": {
			base: "EUR", quote: "JPY", r: fridayToMonday,
			calls:        []*apitest.WorkflowCall{listCurrencies()},
			wantNotFound: true,
		},
		"a history that fails": {
			base: "EUR", quote: "USD", r: fridayToMonday,
			calls: []*apitest.WorkflowCall{listCurrencies(), func() *apitest.WorkflowCall {
				call := history()
				call.Result, call.Err = nil, assert.AnError
				return call
			}()},
			wantErr: "the workflow failed",
		},
		"a range that ends before it starts, without running a workflow": {
			base: "EUR", quote: "USD", r: assetvalue.Range{From: date(2026, 10, 12), To: date(2026, 10, 9)},
			wantErr: "the range ends on 2026-10-09, before it starts on 2026-10-12",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			p, ctx := provider(t, tc.calls...)

			got, err := p.Rates(ctx, tc.base, tc.quote, tc.r)

			switch {
			case tc.wantNotFound:
				var notFound errors.ErrNotFound
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, "Conversion", notFound.ResourceType)
				require.Equal(t, tc.base+"/"+tc.quote, notFound.ID)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
				require.Equal(t, tc.want, got)
			}
		})
	}
}
