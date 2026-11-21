package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerPreviewFinancialAccountDeletion(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "preview-financial-account-deletion",
		Method:      http.MethodGet,
		Path:        "/api/financial-accounts/{id}/deletion-preview",
		Summary:     "Preview deleting a financial account",
		Description: "Reports what DELETE /api/financial-accounts/{id} would do to the account as it is now, and changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *DeletionPreviewInput) (*gateway.AccountDeletionPreviewOutput, error) {
		return gateway.PreviewDeletion(ctx, c, ledgeraccountswf.PreviewAccountDeletion, act.PreviewAccountDeletionInput{FinancialAccountID: &in.ID})
	})
}
