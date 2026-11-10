package holdings

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerListMovements(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-holding-movements",
		Method:      http.MethodGet,
		Path:        "/api/holdings/movements",
		Summary:     "List a page of the movements of holdings, by the date they occurred and then by id",
		Description: "Only the movements that count towards holdings are listed: those without a journal entry, and " +
			"those of a posted one. Their units add up to the holding's.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *ListMovementsInput) (*ListMovementsOutput, error) {
		params := lt.ListPositionMovementsParams{
			FinancialAccountID: in.FinancialAccountID, AssetID: in.AssetID, From: in.From, To: in.To, Limit: in.Limit,
		}
		if in.After != "" {
			occurredAt, id, ok := gateway.DecodeCursor(in.After)
			if !ok {
				return nil, huma.Error422UnprocessableEntity("after is not the next of a page",
					&huma.ErrorDetail{Location: "query.after", Value: in.After, Message: "not the next of a page"})
			}
			params.After = &lt.PositionMovementCursor{OccurredAt: occurredAt, ID: id}
		}
		for _, kind := range in.Kinds {
			params.Kinds = append(params.Kinds, model.PositionMovementKind(kind))
		}

		result, err := gateway.RunWorkflow[lt.ListPositionMovementsResult](ctx, c, portfoliowf.ListPositionMovements, params)
		if err != nil {
			return nil, err
		}
		out := &ListMovementsOutput{Body: MovementPage{Movements: make([]Movement, 0, len(result.Movements))}}
		if result.Next != nil {
			out.Body.Next = gateway.EncodeCursor(result.Next.OccurredAt, result.Next.ID)
		}
		for _, movement := range result.Movements {
			out.Body.Movements = append(out.Body.Movements, newMovement(movement))
		}
		return out, nil
	})
}

// newMovement is movement as the API returns it.
func newMovement(movement model.PositionMovement) Movement {
	out := Movement{
		ID: movement.ID, Kind: string(movement.Kind), FinancialAccountID: movement.FinancialAccountID,
		AssetID: movement.AssetID, UnitsDelta: gateway.NewAmount(movement.UnitsDelta), OccurredAt: movement.OccurredAt,
		JournalEntryID: movement.JournalEntryID,
	}
	if movement.CostBasisAmount != nil && movement.CostBasisCurrencyID != nil {
		out.CostBasis = &CostBasis{CurrencyID: *movement.CostBasisCurrencyID, Amount: gateway.NewAmount(*movement.CostBasisAmount)}
	}
	if movement.Description != nil {
		out.Description = *movement.Description
	}
	if movement.ExternalID != nil {
		out.ExternalID = *movement.ExternalID
	}
	return out
}
