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

func registerRename(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "rename-category",
		Method:      http.MethodPost,
		Path:        "/api/categories/{id}/rename",
		Summary:     "Rename a category",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *RenameCategoryInput) (*struct{}, error) {
		_, err := gateway.RunWorkflow[ct.CategoryResult](ctx, c, categorieswf.RenameCategory,
			ct.RenameCategoryInput{LedgerAccountID: in.ID, Name: in.Body.Name})
		return nil, err
	})
}
