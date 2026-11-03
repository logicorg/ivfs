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

func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-currency",
		Method:      http.MethodPost,
		Path:        "/api/currencies",
		Summary:     "Create a currency, or find the one with its code",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *CreateCurrencyInput) (*CreateCurrencyOutput, error) {
		result, err := gateway.RunWorkflow[ct.InsertCurrencyResult](ctx, c, currencieswf.InsertCurrency, in.Body)
		if err != nil {
			return nil, err
		}
		return &CreateCurrencyOutput{Body: *result}, nil
	})
}
