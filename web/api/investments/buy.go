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

func registerBuy(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "buy-asset",
		Method:      http.MethodPost,
		Path:        "/api/investments/buys",
		Summary:     "Record units of an asset bought with a brokerage account's cash",
		Description: "Cash pays the amount and the fee, Investments receives them, and the units are added at that cost. " +
			"A repeated idempotency key returns the buy first recorded with it, recording nothing. A buy that leaves " +
			"a holding negative at some point in time is recorded all the same, and the response warns about it.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *BuyAssetInput) (*RecordTradeOutput, error) {
		result, err := gateway.RunWorkflow[it.TradeResult](ctx, c, investmentswf.RecordInvestment, it.RecordInvestmentInput{
			Kind: it.KindBuy,
			Buy: &it.BuyAssetInput{
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

// newRecordTradeOutput is a buy's or a sale's result as the API returns it.
func newRecordTradeOutput(result *it.TradeResult) *RecordTradeOutput {
	return &RecordTradeOutput{Body: RecordedTrade{
		JournalEntryID:     result.JournalEntryID,
		PositionMovementID: result.PositionMovementID,
		Warnings:           gateway.NewNegativeHoldings(result.Warnings),
	}}
}
