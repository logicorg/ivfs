package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestCreate checks that the body becomes CreateCategory's input, with or
// without a parent, that its refusals keep their statuses, and that a kind
// other than income or expense, or an empty name, runs no workflow.
func TestCreate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create a category under another": {
			Method: http.MethodPost, Path: "/api/categories",
			Body: `{"name":"Groceries","kind":"expense","currencyId":1,"parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory,
				Params: ct.CreateCategoryInput{
					Name: "Groceries", Kind: model.LedgerAccountExpense, CurrencyID: 1, ParentID: ptr.Int64(10),
				},
				Result: ct.CategoryResult{LedgerAccountID: 11},
			},
			WantStatus: http.StatusOK, WantBody: `{"ledgerAccountId":11}`,
		},
		"create a top-level category": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Salary","kind":"income","currencyId":1}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory,
				Params:   ct.CreateCategoryInput{Name: "Salary", Kind: model.LedgerAccountIncome, CurrencyID: 1},
				Result:   ct.CategoryResult{LedgerAccountID: 20},
			},
			WantStatus: http.StatusOK, WantBody: `{"ledgerAccountId":20}`,
		},
		"create a category under a closed one": {
			Method: http.MethodPost, Path: "/api/categories",
			Body: `{"name":"Groceries","kind":"expense","currencyId":1,"parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory,
				Params: ct.CreateCategoryInput{
					Name: "Groceries", Kind: model.LedgerAccountExpense, CurrencyID: 1, ParentID: ptr.Int64(10),
				},
				Err: apitest.ChildWorkflowError("ErrResourceClosed", "parent ledger account 10 is closed"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "parent ledger account 10 is closed"),
		},
		"create a category in a currency that does not exist": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Salary","kind":"income","currencyId":9}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CreateCategory,
				Params:   ct.CreateCategoryInput{Name: "Salary", Kind: model.LedgerAccountIncome, CurrencyID: 9},
				Err:      apitest.ChildWorkflowError("ErrNotFound", "resource Currency not found with ID: 9"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency not found with ID: 9"),
		},
		"create a category of another kind": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"Cash","kind":"asset","currencyId":1}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"create a category with no name": {
			Method: http.MethodPost, Path: "/api/categories", Body: `{"name":"","kind":"expense","currencyId":1}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
