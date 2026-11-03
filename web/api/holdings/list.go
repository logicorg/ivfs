package holdings

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-holdings",
		Method:      http.MethodGet,
		Path:        "/api/holdings",
		Summary:     "List the units held of each asset in each financial account, and what they cost",
		Description: "The cost basis is replayed from the movements at average cost (ADR 0009). A holding sold to zero " +
			"is listed with no units.",
		Tags: tags,
	}, func(ctx context.Context, in *ListHoldingsInput) (*ListHoldingsOutput, error) {
		result, err := gateway.RunWorkflow[lt.PositionsAtResult](ctx, c, reportingwf.ListPositionsAt, lt.PositionsAtParams{
			FinancialAccountID: in.FinancialAccountID, AssetID: in.AssetID, At: in.At,
		})
		if err != nil {
			return nil, err
		}
		out := &ListHoldingsOutput{Body: HoldingList{Holdings: make([]Holding, 0, len(result.Positions))}}
		for _, position := range result.Positions {
			holding := Holding{
				FinancialAccountID: position.FinancialAccountID, AssetID: position.AssetID,
				Units: gateway.NewAmount(position.Units), CostBasis: make([]CostBasis, 0, len(position.CostBasis)),
				UnknownCostBasis: position.UnknownCostBasis,
			}
			for _, amount := range position.CostBasis {
				holding.CostBasis = append(holding.CostBasis, newCostBasis(amount))
			}
			if position.AverageCost != nil {
				average := newCostBasis(*position.AverageCost)
				holding.AverageCost = &average
			}
			out.Body.Holdings = append(out.Body.Holdings, holding)
		}
		return out, nil
	})
}
