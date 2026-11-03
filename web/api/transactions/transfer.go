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

func registerTransfer(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "record-transfer",
		Method:      http.MethodPost,
		Path:        "/api/transactions/transfers",
		Summary:     "Record money moved from one balance to another, possibly into another currency",
		Description: "Between currencies, toAmount is the amount that arrives, and the transfer is an exchange with no fee. " +
			"A repeated idempotency key returns the entry first recorded with it, posting nothing.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *RecordTransferInput) (*RecordTransactionOutput, error) {
		result, err := gateway.RunWorkflow[tt.RecordResult](ctx, c, transactionswf.RecordTransfer, tt.RecordTransferInput{
			IdempotencyKey:      in.Body.IdempotencyKey,
			FromLedgerAccountID: in.Body.FromLedgerAccountID,
			ToLedgerAccountID:   in.Body.ToLedgerAccountID,
			Amount:              in.Body.Amount.Decimal,
			ToAmount:            in.Body.ToAmount.Decimal,
			OccurredAt:          in.Body.OccurredAt,
			Description:         in.Body.Description,
		})
		if err != nil {
			return nil, err
		}
		return &RecordTransactionOutput{Body: RecordedTransaction{JournalEntryID: result.JournalEntryID}}, nil
	})
}
