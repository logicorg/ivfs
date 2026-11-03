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

func registerUpdate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "update-currency",
		Method:      http.MethodPatch,
		Path:        "/api/currencies/{id}",
		Summary:     "Change a currency's code, name, or symbol",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *UpdateCurrencyInput) (*struct{}, error) {
		params := &ct.UpdateCurrencyParams{ID: in.ID, Code: in.Body.Code, Name: in.Body.Name, Symbol: in.Body.Symbol}
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, currencieswf.UpdateCurrency, params)
	})
}
