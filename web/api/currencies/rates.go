package currencies

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerRates(api huma.API, c client.Client, sources assetvalue.Sources) {
	huma.Register(api, huma.Operation{
		OperationID: "list-conversion-rates",
		Method:      http.MethodGet,
		Path:        "/api/currencies/{id}/rates/{quoteId}",
		Summary:     "List how many units of the quote currency one of the base was worth over a range, from every provider",
		Description: "Every configured provider is asked at once (ADR 0014). One with no rate on any date is " +
			"left out; one that fails, or does not answer in time, is listed in errors. Both lists are in the " +
			"configured order, so the first rates are the default.",
		Tags:   tags,
		Errors: []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *ListConversionRatesInput) (*ListConversionRatesOutput, error) {
		r, err := in.Range()
		if err != nil {
			return nil, err
		}
		if in.ID == in.QuoteID {
			return nil, huma.Error422UnprocessableEntity(errors.ErrInvalidParameter{
				Description: "the base and quote are the same currency, whose rate is always 1"}.Error())
		}
		// Providers speak in codes, and no workflow retrieves a currency by
		// its ID alone.
		listed, err := gateway.RunWorkflow[ct.ListCurrenciesResult](ctx, c, currencieswf.ListCurrencies, ct.ListCurrenciesParams{})
		if err != nil {
			return nil, err
		}
		codes := make(map[int64]string, len(listed.Currencies))
		for _, currency := range listed.Currencies {
			codes[currency.ID] = currency.Code
		}
		for _, id := range []int64{in.ID, in.QuoteID} {
			if _, ok := codes[id]; !ok {
				return nil, huma.Error404NotFound(errors.ErrNotFound{ResourceType: "Currency", ID: id}.Error())
			}
		}
		base, quote := codes[in.ID], codes[in.QuoteID]

		answers := assetvalue.Ask(ctx, sources, func(ctx context.Context, p assetvalue.Provider) ([]assetvalue.Rate, error) {
			return p.Rates(ctx, base, quote, r)
		})
		out := &ListConversionRatesOutput{Body: ConversionRates{Rates: []ProviderRates{}, Errors: []gateway.ProviderError{}}}
		for _, answer := range answers {
			if answer.Failure != "" {
				out.Body.Errors = append(out.Body.Errors, gateway.ProviderError{Provider: answer.Provider, Detail: answer.Failure})
				continue
			}
			out.Body.Rates = append(out.Body.Rates, ProviderRates{Provider: answer.Provider, Rates: conversionRates(answer.Series)})
		}
		return out, nil
	})
}

// conversionRates returns series as the API lists it.
func conversionRates(series []assetvalue.Rate) []ConversionRate {
	rates := make([]ConversionRate, 0, len(series))
	for _, rate := range series {
		rates = append(rates, ConversionRate{
			Date: rate.Date.Format(time.DateOnly), Rate: gateway.NewAmount(rate.Rate), ObservedAt: rate.ObservedAt,
		})
	}
	return rates
}
