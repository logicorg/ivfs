package assets

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/danielgtaylor/huma/v2"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

// provider answers with values, after checking it is asked for apple over
// want, or with err.
type provider struct {
	want   assetvalue.Range
	values []assetvalue.Value
	err    error
}

func (p provider) Values(_ context.Context, asset model.Asset, r assetvalue.Range) ([]assetvalue.Value, error) {
	if asset.Name != "Apple" || *asset.Ticker != "AAPL" || !r.From.Equal(p.want.From) || !r.To.Equal(p.want.To) ||
		r.Interval != p.want.Interval {
		return nil, fmt.Errorf("asked for %s over %v, not Apple over %v", asset.Name, r, p.want)
	}
	return p.values, p.err
}

func (provider) Rates(context.Context, string, string, assetvalue.Range) ([]assetvalue.Rate, error) {
	return nil, errors.ErrNotFound{ResourceType: "Conversion"}
}

// TestValues checks that every provider is asked for the asset over the
// range, that their values are listed with currency IDs in the configured
// order, that one with no value is left out and one that fails, or answers
// in a currency the user lacks, is listed in errors, and that a bad request
// asks no provider.
func TestValues(t *testing.T) {
	friday, saturday := time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	weekend := assetvalue.Range{From: friday, To: saturday, Interval: assetvalue.Daily}
	closedAt := time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC)
	closing := func(date time.Time, currency string) assetvalue.Value {
		return assetvalue.Value{Date: date, Amount: decimal.RequireFromString("231.45"), Currency: currency, ObservedAt: closedAt}
	}

	retrieve := &apitest.WorkflowCall{
		Workflow: assetswf.RetrieveAsset, Params: &at.RetrieveAssetParams{ID: 7},
		Result: at.RetrieveAssetResult{Asset: &model.Asset{ID: 7, Name: "Apple", Kind: model.AssetKindStock, Ticker: new("AAPL")}},
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
			RegisterValues(api, c, assetvalue.Sources{Providers: sources, Timeout: time.Second})
		}
	}

	apitest.RunCases(t, register(weekend,
		assetvalue.Source{Name: "manual", Provider: provider{err: errors.ErrNotFound{ResourceType: "AssetValue"}}},
		assetvalue.Source{Name: "pounds", Provider: provider{values: []assetvalue.Value{closing(friday, "GBP")}}},
		assetvalue.Source{Name: "yahoo", Provider: provider{values: []assetvalue.Value{closing(friday, "USD"), closing(saturday, "usd")}}},
		assetvalue.Source{Name: "broken", Provider: provider{err: fmt.Errorf("call the provider: connection refused")}},
	), map[string]apitest.Case{
		"every provider, over a weekend": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09&to=2026-10-10",
			Calls:      []*apitest.WorkflowCall{retrieve, listCurrencies},
			WantStatus: http.StatusOK,
			WantBody: `{"values":[{"provider":"yahoo","values":[` +
				`{"date":"2026-10-09","value":"231.45","currencyId":2,"observedAt":"2026-10-09T20:00:00Z"},` +
				`{"date":"2026-10-10","value":"231.45","currencyId":2,"observedAt":"2026-10-09T20:00:00Z"}]}],` +
				`"errors":[{"provider":"pounds","detail":"currency GBP is not set up; add it under Currencies"},` +
				`{"provider":"broken","detail":"call the provider: connection refused"}]}`,
		},
		"an asset that does not exist": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09&to=2026-10-10",
			Call: &apitest.WorkflowCall{
				Workflow: assetswf.RetrieveAsset, Params: &at.RetrieveAssetParams{ID: 7},
				Err: temporal.NewApplicationError("resource Asset not found with ID: 7", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Asset not found with ID: 7"),
		},
		"a range that ends before it starts": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-10&to=2026-10-09",
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"invalid parameter: the range ends on 2026-10-09, before it starts on 2026-10-10"),
		},
		"no end": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09",
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a date that is not one": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09&to=tomorrow",
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an interval that is not one": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09&to=2026-10-10&interval=year",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})

	monthly := assetvalue.Range{From: friday, To: saturday.AddDate(0, 1, 0), Interval: assetvalue.Monthly}
	apitest.RunCases(t, register(monthly,
		assetvalue.Source{Name: "manual", Provider: provider{err: errors.ErrNotFound{ResourceType: "AssetValue"}}},
	), map[string]apitest.Case{
		"no provider has a value, so the currencies are not listed": {
			Method: http.MethodGet, Path: "/api/assets/7/values?from=2026-10-09&to=2026-11-10&interval=month",
			Call:       retrieve,
			WantStatus: http.StatusOK, WantBody: `{"values":[],"errors":[]}`,
		},
	})
}
