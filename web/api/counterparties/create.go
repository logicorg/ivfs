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
		result, err := gateway.RunWorkflow[cpt.CreateCounterpartyResult](ctx, c, counterpartieswf.CreateCounterparty,
			cpt.CreateCounterpartyInput{Name: in.Body.Name, ExternalID: in.Body.ExternalID})
		if err != nil {
			return nil, err
		}
		return &CreateCounterpartyOutput{Body: CreatedCounterparty{ID: result.CounterpartyID}}, nil
	})
}
