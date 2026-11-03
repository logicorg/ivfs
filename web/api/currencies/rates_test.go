package currencies

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/client"
)

// provider answers with rates, after checking it is asked for EUR to USD
// over want, or with err.
type provider struct {
	want  assetvalue.Range
	rates []assetvalue.Rate
	err   error
}

func (provider) Values(context.Context, model.Asset, assetvalue.Range) ([]assetvalue.Value, error) {
	return nil, errors.ErrNotFound{ResourceType: "AssetValue"}
}

func (p provider) Rates(_ context.Context, base, quote string, r assetvalue.Range) ([]assetvalue.Rate, error) {
	if base != "EUR" || quote != "USD" || !r.From.Equal(p.want.From) || !r.To.Equal(p.want.To) ||
		r.Interval != p.want.Interval {
		return nil, fmt.Errorf("asked for %s/%s over %v, not EUR/USD over %v", base, quote, r, p.want)
	}
	return p.rates, p.err
}

// TestRates checks that every provider is asked for the pair's codes over
// the range, that their rates are listed in the configured order, that one
// with no rate is left out and one that fails is listed in errors, and that
// a bad request asks no provider.
func TestRates(t *testing.T) {
	friday, saturday := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	weekend := assetvalue.Range{From: friday, To: saturday, Interval: assetvalue.Daily}
	closedAt := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)
	rate := func(date time.Time) assetvalue.Rate {
		return assetvalue.Rate{Date: date, Rate: decimal.RequireFromString("1.0842"), ObservedAt: closedAt}
	}

	listCurrencies := &apitest.WorkflowCall{
		Workflow: currencieswf.ListCurrencies, Params: ct.ListCurrenciesParams{},
		Result: ct.ListCurrenciesResult{Currencies: []*model.Currency{{ID: 1, Code: "EUR"}, {ID: 2, Code: "USD"}}},
	}
	register := func(r assetvalue.Range, sources ...assetvalue.Source) apitest.Register {
		for i := range sources {
			p := sources[i].Provider.(provider)
			p.want = r
			sources[i].Provider = p
		}
		return func(api huma.API, c client.Client) {
			RegisterRates(api, c, assetvalue.Sources{Providers: sources, Timeout: time.Second})
		}
	}

	apitest.RunCases(t, register(weekend,
		assetvalue.Source{Name: "manual", Provider: provider{err: errors.ErrNotFound{ResourceType: "Conversion"}}},
		assetvalue.Source{Name: "yahoo", Provider: provider{rates: []assetvalue.Rate{rate(friday), rate(saturday)}}},
		assetvalue.Source{Name: "broken", Provider: provider{err: fmt.Errorf("call the provider: connection refused")}},
	), map[string]apitest.Case{
		"every provider, over a weekend": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/2?from=2026-10-09&to=2026-10-10",
			Call:       listCurrencies,
			WantStatus: http.StatusOK,
			WantBody: `{"rates":[{"provider":"yahoo","rates":[` +
				`{"date":"2026-10-09","rate":"1.0842","observedAt":"2026-10-09T20:00:00Z"},` +
				`{"date":"2026-10-10","rate":"1.0842","observedAt":"2026-10-09T20:00:00Z"}]}],` +
				`"errors":[{"provider":"broken","detail":"call the provider: connection refused"}]}`,
		},
		"a base that does not exist": {
			Method: http.MethodGet, Path: "/api/currencies/9/rates/2?from=2026-10-09&to=2026-10-10",
			Call:       listCurrencies,
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency not found with ID: 9"),
		},
		"a quote that does not exist": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/9?from=2026-10-09&to=2026-10-10",
			Call:       listCurrencies,
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency not found with ID: 9"),
		},
		"the same currency": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/1?from=2026-10-09&to=2026-10-10",
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"invalid parameter: the base and quote are the same currency, whose rate is always 1"),
		},
		"a range that ends before it starts": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/2?from=2026-10-10&to=2026-10-09",
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"invalid parameter: the range ends on 2026-10-09, before it starts on 2026-10-10"),
		},
		"an interval that is not one": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/2?from=2026-10-09&to=2026-10-10&interval=year",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})

	monthly := assetvalue.Range{From: friday, To: saturday.AddDate(0, 1, 0), Interval: assetvalue.Monthly}
	apitest.RunCases(t, register(monthly,
		assetvalue.Source{Name: "manual", Provider: provider{err: errors.ErrNotFound{ResourceType: "Conversion"}}},
	), map[string]apitest.Case{
		"no provider has a rate": {
			Method: http.MethodGet, Path: "/api/currencies/1/rates/2?from=2026-10-09&to=2026-11-10&interval=month",
			Call:       listCurrencies,
			WantStatus: http.StatusOK, WantBody: `{"rates":[],"errors":[]}`,
		},
	})
}
