package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDelete checks that deleting runs DeleteCategory under the category's
// deletion ID, with the replacement if one is asked for, and that its refusals
// keep their statuses.
func TestDelete(t *testing.T) {
	remove := func(params ct.DeleteCategoryInput, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: categorieswf.DeleteCategory, Params: params, Result: lt.DeleteAccountResult{}, Err: err,
			WorkflowID: workflows.DeleteCategoryWorkflowID(params.LedgerAccountID),
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete it": {
			Method: http.MethodDelete, Path: "/api/categories/11",
			Call:       remove(ct.DeleteCategoryInput{LedgerAccountID: 11}, nil),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"merge it into a replacement": {
			Method: http.MethodDelete, Path: "/api/categories/11?replacementId=12",
			Call: remove(ct.DeleteCategoryInput{LedgerAccountID: 11, ReplacementID: ptr.Int64(12)},
				apitest.ChildWorkflowError("ErrInvalidParameter", "merging an account into a replacement is not supported yet; delete it without a replacement")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"merging an account into a replacement is not supported yet; delete it without a replacement"),
		},
		"an account that is not a category": {
			Method: http.MethodDelete, Path: "/api/categories/7",
			Call: remove(ct.DeleteCategoryInput{LedgerAccountID: 7},
				temporal.NewApplicationError("ledger account 7 has kind asset, not income or expense; delete it as a ledger account", "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"ledger account 7 has kind asset, not income or expense; delete it as a ledger account"),
		},
		"a category with categories under it": {
			Method: http.MethodDelete, Path: "/api/categories/10",
			Call: remove(ct.DeleteCategoryInput{LedgerAccountID: 10}, apitest.ChildWorkflowError("ErrResourceConflict",
				"resource LedgerAccount conflict: ledger account 10 has child accounts; delete or move them first")),
			WantStatus: http.StatusConflict,
			WantBody: apitest.Problem(http.StatusConflict,
				"resource LedgerAccount conflict: ledger account 10 has child accounts; delete or move them first"),
		},
		"id 0": {
			Method: http.MethodDelete, Path: "/api/categories/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
