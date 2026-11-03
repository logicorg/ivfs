package transactions

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	transactionswf "github.com/logicorg/ivfs/finance/workflows/facade/transactions"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerIncome(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "record-income",
		Method:      http.MethodPost,
		Path:        "/api/transactions/incomes",
		Summary:     "Record money earned on an income category into a balance",
		Description: "A repeated idempotency key returns the entry first recorded with it, posting nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *RecordIncomeInput) (*RecordTransactionOutput, error) {
		result, err := gateway.RunWorkflow[tt.RecordResult](ctx, c, transactionswf.RecordIncome, tt.RecordIncomeInput{
			IdempotencyKey:          in.Body.IdempotencyKey,
			BalanceLedgerAccountID:  in.Body.BalanceLedgerAccountID,
			CategoryLedgerAccountID: in.Body.CategoryLedgerAccountID,
			Amount:                  in.Body.Amount.Decimal,
			OccurredAt:              in.Body.OccurredAt,
			Description:             in.Body.Description,
		})
		if err != nil {
			return nil, err
		}
		return &RecordTransactionOutput{Body: RecordedTransaction{JournalEntryID: result.JournalEntryID}}, nil
	})
}
