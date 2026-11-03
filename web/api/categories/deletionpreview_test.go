package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestDeletionPreview checks that the preview runs PreviewCategoryDeletion
// and returns its preview, and that a balance is refused.
func TestDeletionPreview(t *testing.T) {
	preview := func(result lt.AccountDeletionPreview, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: categorieswf.PreviewCategoryDeletion, Params: ct.PreviewCategoryDeletionInput{LedgerAccountID: 11},
			Result: result, Err: err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"preview it": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call: preview(lt.AccountDeletionPreview{
				DeletedEntries: 1, KeptEntries: 4,
				StrayAmounts: []lt.StrayAmount{{Name: "Uncategorized", Kind: model.LedgerAccountExpense, CurrencyCode: "EUR", Amount: decimal.NewFromInt(42)}},
			}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"deletedEntries":1,"keptEntries":4,"deletedMovementGroups":0,"keptMovementGroups":0,` +
				`"strayAmounts":[{"name":"Uncategorized","kind":"expense","currencyCode":"EUR","amount":"42"}]}`,
		},
		"a balance": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call: preview(lt.AccountDeletionPreview{}, temporal.NewApplicationError(
				"ledger account 11 has kind asset, not income or expense; delete it as a ledger account", "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"ledger account 11 has kind asset, not income or expense; delete it as a ledger account"),
		},
		"a category that does not exist": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call:       preview(lt.AccountDeletionPreview{}, temporal.NewApplicationError("resource LedgerAccount not found with ID: 11", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 11"),
		},
		"id 0": {
			Method: http.MethodGet, Path: "/api/categories/0/deletion-preview", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
