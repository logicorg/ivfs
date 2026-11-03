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

func registerMove(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "move-category",
		Method:      http.MethodPost,
		Path:        "/api/categories/{id}/move",
		Summary:     "Move a category under another, or to the top level",
		Description: "A category can move only while it and the categories under it have no postings.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *MoveCategoryInput) (*struct{}, error) {
		_, err := gateway.RunWorkflow[ct.CategoryResult](ctx, c, categorieswf.MoveCategory,
			ct.MoveCategoryInput{LedgerAccountID: in.ID, ParentID: in.Body.ParentID})
		return nil, err
	})
}
