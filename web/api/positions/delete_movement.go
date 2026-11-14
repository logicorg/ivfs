package positions

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDeleteMovement(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-position-movement",
		Method:      http.MethodDelete,
		Path:        "/api/position-movements/{id}",
		Summary:     "Delete a position movement with the rest of its group",
		Description: "Any movement of a split, a transfer, or an adjustment deletes all of it, and its idempotency key is " +
			"free again. A movement of a journal entry, a trade or a vest, is a 422: it is deleted with its entry. " +
			"Deleting a movement that does not exist changes nothing. A deletion that leaves a holding negative at " +
			"some point in time is made all the same, and the response warns about it.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *DeletePositionMovementInput) (*DeletePositionMovementOutput, error) {
		result, err := gateway.RunWorkflow[lt.DeletePositionMovementResult](ctx, c, portfoliowf.DeletePositionMovement,
			lt.DeletePositionMovementInput{MovementID: in.ID})
		if err != nil {
			return nil, err
		}
		return &DeletePositionMovementOutput{Body: gateway.Deletion{Warnings: gateway.NewNegativeHoldings(result.Warnings)}}, nil
	})
}
