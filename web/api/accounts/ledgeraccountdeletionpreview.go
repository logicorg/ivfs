package accounts

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerPreviewLedgerAccountDeletion(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "preview-ledger-account-deletion",
		Method:      http.MethodGet,
		Path:        "/api/ledger-accounts/{id}/deletion-preview",
		Summary:     "Preview deleting a ledger account",
		Description: "Reports what DELETE /api/ledger-accounts/{id} would do to the account as it is now, and changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *DeletionPreviewInput) (*gateway.AccountDeletionPreviewOutput, error) {
		return gateway.PreviewDeletion(ctx, c, ledgeraccountswf.PreviewAccountDeletion, lt.PreviewAccountDeletionInput{LedgerAccountID: &in.ID})
	})
}
