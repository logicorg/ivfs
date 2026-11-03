package categories

import (
	"net/http"
	"testing"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that categories are listed with their currency's code,
// that the closed filter reaches ListCategories, and that a category without
// its currency is a 500 rather than a code left empty.
func TestList(t *testing.T) {
	euro := &model.Currency{ID: 1, Code: "EUR"}
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	food := model.LedgerAccount{ID: 10, Name: "Food", Kind: model.LedgerAccountExpense, CurrencyID: 1, Currency: euro}
	groceries := model.LedgerAccount{
		ID: 11, Name: "Groceries", Kind: model.LedgerAccountExpense, CurrencyID: 1, Currency: euro, ParentID: ptr.Int64(10),
	}
	salary := model.LedgerAccount{
		ID: 20, Name: "Salary", Kind: model.LedgerAccountIncome, CurrencyID: 1, Currency: euro, ClosedAt: &closedAt,
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"list the open categories": {
			Method: http.MethodGet, Path: "/api/categories",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{},
				Result: ct.ListCategoriesResult{LedgerAccounts: []model.LedgerAccount{food, groceries}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"categories":[` +
				`{"id":10,"name":"Food","kind":"expense","currencyId":1,"currencyCode":"EUR","closed":false},` +
				`{"id":11,"name":"Groceries","kind":"expense","currencyId":1,"currencyCode":"EUR","parentId":10,"closed":false}]}`,
		},
		"list the closed categories too": {
			Method: http.MethodGet, Path: "/api/categories?includeClosed=true",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{IncludeClosed: true},
				Result: ct.ListCategoriesResult{LedgerAccounts: []model.LedgerAccount{salary}},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"categories":[{"id":20,"name":"Salary","kind":"income","currencyId":1,"currencyCode":"EUR","closed":true}]}`,
		},
		"list no categories": {
			Method: http.MethodGet, Path: "/api/categories",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{},
				Result: ct.ListCategoriesResult{},
			},
			WantStatus: http.StatusOK, WantBody: `{"categories":[]}`,
		},
		"list a category without its currency": {
			Method: http.MethodGet, Path: "/api/categories",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.ListCategories, Params: ct.ListCategoriesParams{},
				Result: ct.ListCategoriesResult{LedgerAccounts: []model.LedgerAccount{{ID: 10, Name: "Food"}}},
			},
			WantStatus: http.StatusInternalServerError,
		},
	})
}
