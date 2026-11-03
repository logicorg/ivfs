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

func registerSell(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "sell-asset",
		Method:      http.MethodPost,
		Path:        "/api/investments/sells",
		Summary:     "Record units of an asset sold for a brokerage account's cash",
		Description: "Investments pays the amount less the fee and Cash receives it, and the units are removed. The sale " +
			"stores no basis: the average cost it removes, and its gain, are worked out when read. A repeated " +
			"idempotency key returns the sale first recorded with it, recording nothing. A sale of more units than " +
			"are held is recorded all the same, and the response warns about the negative holding.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *SellAssetInput) (*RecordTradeOutput, error) {
		result, err := gateway.RunWorkflow[it.TradeResult](ctx, c, investmentswf.RecordInvestment, it.RecordInvestmentInput{
			Kind: it.KindSell,
			Sell: &it.SellAssetInput{
				IdempotencyKey:     in.Body.IdempotencyKey,
				FinancialAccountID: in.Body.FinancialAccountID,
				AssetID:            in.Body.AssetID,
				Units:              in.Body.Units.Decimal,
				Amount:             in.Body.Amount.Decimal,
				Fee:                in.Body.Fee.Decimal,
				CurrencyCode:       in.Body.CurrencyCode,
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
