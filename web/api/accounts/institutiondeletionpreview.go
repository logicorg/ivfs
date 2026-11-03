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

func registerPreviewInstitutionDeletion(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "preview-institution-deletion",
		Method:      http.MethodGet,
		Path:        "/api/institutions/{id}/deletion-preview",
		Summary:     "Preview deleting an institution",
		Description: "Reports what DELETE /api/institutions/{id} would do to its financial accounts as they are now, and changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound},
	}, func(ctx context.Context, in *DeletionPreviewInput) (*gateway.AccountDeletionPreviewOutput, error) {
		return gateway.PreviewDeletion(ctx, c, ledgeraccountswf.PreviewAccountDeletion, lt.PreviewAccountDeletionInput{InstitutionID: &in.ID})
	})
}
