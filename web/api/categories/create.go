package categories

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-category",
		Method:      http.MethodPost,
		Path:        "/api/categories",
		Summary:     "Create a category, or find the one with its name under the same parent",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *CreateCategoryInput) (*CreateCategoryOutput, error) {
		result, err := gateway.RunWorkflow[ct.CategoryResult](ctx, c, categorieswf.CreateCategory, ct.CreateCategoryInput{
			Name: in.Body.Name, Kind: model.LedgerAccountKind(in.Body.Kind),
			CurrencyID: in.Body.CurrencyID, ParentID: in.Body.ParentID,
		})
		if err != nil {
			return nil, err
		}
		return &CreateCategoryOutput{Body: *result}, nil
	})
}
