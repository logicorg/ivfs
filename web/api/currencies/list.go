package currencies

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-currencies",
		Method:      http.MethodGet,
		Path:        "/api/currencies",
		Summary:     "List every currency",
		Tags:        tags,
	}, func(ctx context.Context, _ *struct{}) (*ListCurrenciesOutput, error) {
		result, err := gateway.RunWorkflow[ct.ListCurrenciesResult](ctx, c, currencieswf.ListCurrencies, ct.ListCurrenciesParams{})
		if err != nil {
			return nil, err
		}
		out := &ListCurrenciesOutput{Body: CurrencyList{Currencies: make([]Currency, 0, len(result.Currencies))}}
		for _, currency := range result.Currencies {
			out.Body.Currencies = append(out.Body.Currencies, Currency{
				ID: currency.ID, Code: currency.Code, Name: currency.Name,
				Symbol: currency.Symbol, Precision: currency.Precision,
			})
		}
		return out, nil
	})
}
