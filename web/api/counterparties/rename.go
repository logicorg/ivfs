package counterparties

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	counterpartieswf "github.com/logicorg/ivfs/finance/workflows/ledger/counterparties"
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
		_, err := gateway.RunWorkflow[cpt.CounterpartyMutationResult](ctx, c, counterpartieswf.RenameCounterparty,
			cpt.RenameCounterpartyInput{CounterpartyID: in.ID, Name: in.Body.Name})
		return nil, err
	})
}
