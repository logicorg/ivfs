package assets

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerValues(api huma.API, c client.Client, sources assetvalue.Sources) {
	huma.Register(api, huma.Operation{
		OperationID: "list-asset-values",
		Method:      http.MethodGet,
		Path:        "/api/assets/{id}/values",
		Summary:     "List the asset's values over a range, from every provider",
		Description: "Every configured provider is asked at once (ADR 0014). One with no value on any date is " +
			"left out; one that fails, or does not answer in time, is listed in errors. Both lists are in the " +
			"configured order, so the first values are the default.",
		Tags:   tags,
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *ListAssetValuesInput) (*ListAssetValuesOutput, error) {
		r, err := in.Range()
		if err != nil {
			return nil, err
		}
		retrieved, err := gateway.RunWorkflow[at.RetrieveAssetResult](ctx, c, assetswf.RetrieveAsset, &at.RetrieveAssetParams{ID: in.ID})
		if err != nil {
			return nil, err
		}
		asset := *retrieved.Asset

		answers := assetvalue.Ask(ctx, sources, func(ctx context.Context, p assetvalue.Provider) ([]assetvalue.Value, error) {
			return p.Values(ctx, asset, r)
		})
		out := &ListAssetValuesOutput{Body: AssetValues{Values: []ProviderValues{}, Errors: []gateway.ProviderError{}}}
		var ids map[string]int64
		for _, answer := range answers {
			if answer.Failure != "" {
				out.Body.Errors = append(out.Body.Errors, gateway.ProviderError{Provider: answer.Provider, Detail: answer.Failure})
				continue
			}
			// Listed once, and only when a provider has values to map.
			if ids == nil {
				if ids, err = currencyIDs(ctx, c); err != nil {
					return nil, err
				}
			}
			values, missing := assetValues(answer.Series, ids)
			if missing != "" {
				out.Body.Errors = append(out.Body.Errors, gateway.ProviderError{
					Provider: answer.Provider,
					Detail:   fmt.Sprintf("currency %s is not set up; add it under Currencies", missing),
				})
				continue
			}
			out.Body.Values = append(out.Body.Values, ProviderValues{Provider: answer.Provider, Values: values})
		}
		return out, nil
	})
}

// currencyIDs maps the codes of the user's currencies to their IDs: a
// provider speaks in codes, the API in IDs.
func currencyIDs(ctx context.Context, c client.Client) (map[string]int64, error) {
	result, err := gateway.RunWorkflow[ct.ListCurrenciesResult](ctx, c, currencieswf.ListCurrencies, ct.ListCurrenciesParams{})
	if err != nil {
		return nil, err
	}
	ids := make(map[string]int64, len(result.Currencies))
	for _, currency := range result.Currencies {
		ids[currency.Code] = currency.ID
	}
	return ids, nil
}

// assetValues returns series as the API lists it, or the code of the first
// currency in it the user has not set up.
func assetValues(series []assetvalue.Value, ids map[string]int64) ([]AssetValue, string) {
	values := make([]AssetValue, 0, len(series))
	for _, value := range series {
		// Codes are stored upper-cased; a provider is not trusted to be.
		id, ok := ids[strings.ToUpper(value.Currency)]
		if !ok {
			return nil, value.Currency
		}
		values = append(values, AssetValue{
			Date: value.Date.Format(time.DateOnly), Value: gateway.NewAmount(value.Amount),
			CurrencyID: id, ObservedAt: value.ObservedAt,
		})
	}
	return values, ""
}
