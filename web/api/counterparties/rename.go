package counterparties

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerRename(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "rename-counterparty",
		Method:      http.MethodPost,
		Path:        "/api/counterparties/{id}/rename",
		Summary:     "Rename a counterparty",
		Description: "The journal entries that name it show the new name.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *RenameCounterpartyInput) (*struct{}, error) {
		_, err := gateway.RunWorkflow[lt.CounterpartyMutationResult](ctx, c, ledgeraccountswf.RenameCounterparty,
			lt.RenameCounterpartyInput{CounterpartyID: in.ID, Name: in.Body.Name})
		return nil, err
	})
}
