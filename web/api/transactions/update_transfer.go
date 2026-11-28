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

func registerUpdateTransfer(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "update-transfer",
		Method:      http.MethodPut,
		Path:        "/api/transactions/transfers/{id}",
		Summary:     "Correct a journal entry to a transfer, possibly into another currency",
		Description: "Between currencies, the rate is worked out again from the amounts, and a provider reference the " +
			"entry had is dropped. " + correctionDescription,
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *UpdateTransferInput) (*UpdateTransactionOutput, error) {
		result, err := gateway.RunWorkflow[tt.UpdateResult](ctx, c, transactionswf.UpdateTransfer, tt.UpdateTransferInput{
			JournalEntryID:      in.ID,
			FromLedgerAccountID: in.Body.FromLedgerAccountID,
			ToLedgerAccountID:   in.Body.ToLedgerAccountID,
			Amount:              in.Body.Amount.Decimal,
			ToAmount:            in.Body.ToAmount.Decimal,
			OccurredAt:          in.Body.OccurredAt,
			Description:         in.Body.Description,
			TagIDs:              in.Body.TagIDs,
			Charges:             splits(in.Body.Charges),
		})
		if err != nil {
			return nil, err
		}
		return updated(result), nil
	})
}
