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

func registerClose(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "close-category",
		Method:      http.MethodPost,
		Path:        "/api/categories/{id}/close",
		Summary:     "Close a category",
		Description: "A category with open categories under it cannot be closed until they are. Closing keeps its history; to remove it, delete it.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusConflict},
	}, func(ctx context.Context, in *CloseCategoryInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, categorieswf.CloseCategory,
			ct.CloseCategoryInput{LedgerAccountID: in.ID})
	})
}
