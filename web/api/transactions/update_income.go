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

func registerUpdateIncome(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "update-income",
		Method:      http.MethodPut,
		Path:        "/api/transactions/incomes/{id}",
		Summary:     "Correct a journal entry to an income",
		Description: correctionDescription,
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity, http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *UpdateIncomeInput) (*UpdateTransactionOutput, error) {
		result, err := gateway.RunWorkflow[tt.UpdateResult](ctx, c, transactionswf.UpdateIncome, tt.UpdateIncomeInput{
			JournalEntryID:         in.ID,
			BalanceLedgerAccountID: in.Body.BalanceLedgerAccountID,
			CategoryID:             in.Body.CategoryID,
			Amount:                 in.Body.Amount.Decimal,
			OccurredAt:             in.Body.OccurredAt,
			Description:            in.Body.Description,
			TagIDs:                 in.Body.TagIDs,
		})
		if err != nil {
			return nil, err
		}
		return updated(result), nil
	})
}
