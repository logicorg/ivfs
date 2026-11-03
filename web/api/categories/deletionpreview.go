package categories

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDeletionPreview(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "preview-category-deletion",
		Method:      http.MethodGet,
		Path:        "/api/categories/{id}/deletion-preview",
		Summary:     "Preview deleting a category",
		Description: "Reports what DELETE /api/categories/{id} would do to the category as it is now, and changes nothing.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *DeletionPreviewInput) (*gateway.AccountDeletionPreviewOutput, error) {
		return gateway.PreviewDeletion(ctx, c, categorieswf.PreviewCategoryDeletion, ct.PreviewCategoryDeletionInput{LedgerAccountID: in.ID})
	})
}
