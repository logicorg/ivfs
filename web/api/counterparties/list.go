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

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-counterparties",
		Method:      http.MethodGet,
		Path:        "/api/counterparties",
		Summary:     "List the counterparties, sorted by name",
		Tags:        tags,
	}, func(ctx context.Context, in *ListCounterpartiesInput) (*ListCounterpartiesOutput, error) {
		result, err := gateway.RunWorkflow[lt.ListCounterpartiesResult](ctx, c, ledgeraccountswf.ListCounterparties,
			lt.ListCounterpartiesParams{NameContains: in.NameContains})
		if err != nil {
			return nil, err
		}
		out := &ListCounterpartiesOutput{Body: CounterpartyList{Counterparties: make([]Counterparty, 0, len(result.Counterparties))}}
		for _, counterparty := range result.Counterparties {
			out.Body.Counterparties = append(out.Body.Counterparties, Counterparty{
				ID: counterparty.ID, Name: counterparty.Name, ExternalID: counterparty.ExternalID,
			})
		}
		return out, nil
	})
}
