package categories

import (
	"net/http"
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that categories are listed with their parent and whether
// they are closed, and that the closed filter reaches ListCategories.
func TestList(t *testing.T) {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	food := model.Category{ID: 10, Name: "Food", Kind: model.LedgerAccountExpense}
	groceries := model.Category{ID: 11, Name: "Groceries", Kind: model.LedgerAccountExpense, ParentID: ptr.Int64(10)}
	salary := model.Category{ID: 20, Name: "Salary", Kind: model.LedgerAccountIncome, ClosedAt: &closedAt}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"list the open categories": {
			Method: http.MethodGet, Path: "/api/categories",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{},
				Result: ct.ListCategoriesResult{Categories: []model.Category{food, groceries}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"categories":[` +
				`{"id":10,"name":"Food","kind":"expense","closed":false},` +
				`{"id":11,"name":"Groceries","kind":"expense","parentId":10,"closed":false}]}`,
		},
		"list the closed categories too": {
			Method: http.MethodGet, Path: "/api/categories?includeClosed=true",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{IncludeClosed: true},
				Result: ct.ListCategoriesResult{Categories: []model.Category{salary}},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"categories":[{"id":20,"name":"Salary","kind":"income","closed":true}]}`,
		},
		"list no categories": {
			Method: http.MethodGet, Path: "/api/categories",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{},
				Result: ct.ListCategoriesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"categories":[]}`,
		},
	})
}
