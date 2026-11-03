package categories

import (
	"context"
	"fmt"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
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
		out := &ListCategoriesOutput{Body: CategoryList{Categories: make([]Category, 0, len(result.LedgerAccounts))}}
		for _, account := range result.LedgerAccounts {
			if account.Currency == nil {
				// ListLedgerAccounts always loads it, so this is a bug.
				return nil, gateway.WorkflowError(fmt.Errorf("ledger account %d has no currency loaded", account.ID))
			}
			out.Body.Categories = append(out.Body.Categories, Category{
				ID: account.ID, Name: account.Name, Kind: string(account.Kind),
				CurrencyID: account.CurrencyID, CurrencyCode: account.Currency.Code,
				ParentID: account.ParentID, Closed: account.ClosedAt != nil,
			})
		}
		return out, nil
	})
}
