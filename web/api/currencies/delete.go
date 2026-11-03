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

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-currency",
		Method:      http.MethodDelete,
		Path:        "/api/currencies/{id}",
		Summary:     "Delete a currency that nothing uses",
		Description: "A currency that anything, such as an account, still uses cannot be deleted. Deleting a currency that does not exist changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *DeleteCurrencyInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, currencieswf.DeleteCurrency, &ct.DeleteCurrencyParams{ID: in.ID})
	})
}
