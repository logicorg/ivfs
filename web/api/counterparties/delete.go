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

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-counterparty",
		Method:      http.MethodDelete,
		Path:        "/api/counterparties/{id}",
		Summary:     "Delete a counterparty",
		Description: "The journal entries that name it keep their postings and lose the counterparty, and its name can be " +
			"used again. Deleting a counterparty that does not exist changes nothing. Entries that keep being recorded " +
			"with it while it is deleted are answered with a conflict, and deleting again is safe.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *DeleteCounterpartyInput) (*DeleteCounterpartyOutput, error) {
		result, err := gateway.RunWorkflow[lt.DeleteCounterpartyResult](ctx, c, ledgeraccountswf.DeleteCounterparty,
			lt.DeleteCounterpartyInput{CounterpartyID: in.ID})
		if err != nil {
			return nil, err
		}
		return &DeleteCounterpartyOutput{Body: DeletedCounterparty{Cleared: result.Cleared}}, nil
	})
}
