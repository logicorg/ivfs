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

func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-counterparty",
		Method:      http.MethodPost,
		Path:        "/api/counterparties",
		Summary:     "Create a counterparty, or find the one with its name",
		Description: "Names are compared ignoring case and surrounding space. A counterparty that exists is returned unchanged, whatever its external id.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *CreateCounterpartyInput) (*CreateCounterpartyOutput, error) {
		result, err := gateway.RunWorkflow[lt.CreateCounterpartyResult](ctx, c, ledgeraccountswf.CreateCounterparty,
			lt.CreateCounterpartyInput{Name: in.Body.Name, ExternalID: in.Body.ExternalID})
		if err != nil {
			return nil, err
		}
		return &CreateCounterpartyOutput{Body: CreatedCounterparty{ID: result.CounterpartyID}}, nil
	})
}
