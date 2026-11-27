package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCreate checks that the body becomes CreateCategory's input, with or
// without a parent, that its refusals keep their statuses, and that a kind
// other than income or expense, or an empty name, runs no workflow.
func TestCreate(t *testing.T) {
	groceries := ct.CreateCategoryInput{Name: "Groceries", Kind: model.LedgerAccountExpense, ParentID: ptr.Int64(10)}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create a category under another": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Groceries","kind":"expense","parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory, Params: groceries, Result: ct.CategoryResult{CategoryID: 11},
			},
			WantStatus: http.StatusOK, WantBody: `{"categoryId":11}`,
		},
		"create a top-level category": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Salary","kind":"income"}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory,
				Params:   ct.CreateCategoryInput{Name: "Salary", Kind: model.LedgerAccountIncome},
				Result:   ct.CategoryResult{CategoryID: 20},
			},
			WantStatus: http.StatusOK, WantBody: `{"categoryId":20}`,
		},
		"create a category under a closed one": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Groceries","kind":"expense","parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory, Params: groceries,
				Err: temporal.NewApplicationError("category 10 is closed", "ErrResourceClosed"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "category 10 is closed"),
		},
		"create a category under one that does not exist": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Groceries","kind":"expense","parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory, Params: groceries,
				Err: temporal.NewApplicationError("resource Category not found with ID: 10", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Category not found with ID: 10"),
		},
		"create a category of another kind": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Cash","kind":"asset"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"create a category with no name": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"","kind":"expense"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"create a category with a currency, which it no longer has": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Salary","kind":"income","currencyId":1}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
