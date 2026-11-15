package investments

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	investmentswf "github.com/logicorg/ivfs/finance/workflows/facade/investments"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerVest(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "vest-asset",
		Method:      http.MethodPost,
		Path:        "/api/investments/vests",
		Summary:     "Record units of an asset received as compensation in a brokerage account",
		Description: "Income is credited with the value of every unit vested, under the income category; Investments is " +
			"debited with that of the units delivered, and Expenses with that of the units withheld, under the tax category. " +
			"The delivered units are added at their value. " +
			"Shares sold to cover the tax are a separate sale. A repeated idempotency key returns the vest first recorded " +
			"with it, recording nothing.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *VestAssetInput) (*RecordTradeOutput, error) {
		result, err := gateway.RunWorkflow[it.TradeResult](ctx, c, investmentswf.RecordInvestment, it.RecordInvestmentInput{
			Kind: it.KindVest,
			Vest: &it.VestInput{
				IdempotencyKey:     in.Body.IdempotencyKey,
				FinancialAccountID: in.Body.FinancialAccountID,
				AssetID:            in.Body.AssetID,
				UnitsVested:        in.Body.UnitsVested.Decimal,
				UnitsWithheld:      in.Body.UnitsWithheld.Decimal,
				FairMarketValue:    in.Body.FairMarketValue.Decimal,
				CurrencyCode:       in.Body.CurrencyCode,
				IncomeCategoryID:   in.Body.IncomeCategoryID,
				TaxCategoryID:      in.Body.TaxCategoryID,
				OccurredAt:         in.Body.OccurredAt,
				Description:        in.Body.Description,
			},
		})
		if err != nil {
			return nil, err
		}
		return newRecordTradeOutput(result), nil
	})
}
