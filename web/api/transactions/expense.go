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

func registerExpense(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "record-expense",
		Method:      http.MethodPost,
		Path:        "/api/transactions/expenses",
		Summary:     "Record money spent from a balance on an expense category",
		Description: "A repeated idempotency key returns the entry first recorded with it, posting nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *RecordExpenseInput) (*RecordTransactionOutput, error) {
		result, err := gateway.RunWorkflow[tt.RecordResult](ctx, c, transactionswf.RecordExpense, tt.RecordExpenseInput{
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
