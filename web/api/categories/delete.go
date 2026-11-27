package categories

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerDelete(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "delete-category",
		Method:      http.MethodDelete,
		Path:        "/api/categories/{id}",
		Summary:     "Delete a category, or merge it into another",
		Description: "No entry is deleted and no balance moves: the entries filed under the category lose it, or take the " +
			"replacement. A category with categories under it cannot be deleted until they are. Deleting a category that " +
			"does not exist changes nothing.",
		Tags:   tags,
		Errors: []int{http.StatusConflict, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *DeleteCategoryInput) (*struct{}, error) {
		return nil, gateway.RunWorkflowWithoutResult(ctx, c, categorieswf.DeleteCategory,
			ct.DeleteCategoryInput{CategoryID: in.ID, ReplacementID: gateway.Replacement(in.ReplacementID)})
	})
}
