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

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-categories",
		Method:      http.MethodGet,
		Path:        "/api/categories",
		Summary:     "List the income and expense categories",
		Tags:        tags,
	}, func(ctx context.Context, in *ListCategoriesInput) (*ListCategoriesOutput, error) {
		result, err := gateway.RunWorkflow[ct.ListCategoriesResult](ctx, c, categorieswf.ListCategories,
			ct.ListCategoriesParams{IncludeClosed: in.IncludeClosed})
		if err != nil {
			return nil, err
		}
		out := &ListCategoriesOutput{Body: CategoryList{Categories: make([]Category, 0, len(result.Categories))}}
		for _, category := range result.Categories {
			out.Body.Categories = append(out.Body.Categories, Category{
				ID: category.ID, Name: category.Name, Kind: string(category.Kind),
				ParentID: category.ParentID, Closed: category.ClosedAt != nil,
			})
		}
		return out, nil
	})
}
