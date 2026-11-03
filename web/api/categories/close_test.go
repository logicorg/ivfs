package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestClose checks that closing runs CloseCategory with no closing
// date, so the activity closes it now, and that its refusals keep their
// statuses.
func TestClose(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"close a category": {
			Method: http.MethodPost, Path: "/api/categories/11/close",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CloseCategory,
				Params:   ct.CloseCategoryInput{LedgerAccountID: 11},
			},
			WantStatus: http.StatusNoContent,
		},
		"close a category with open categories under it": {
			Method: http.MethodPost, Path: "/api/categories/10/close",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CloseCategory,
				Params:   ct.CloseCategoryInput{LedgerAccountID: 10},
				Err:      apitest.ChildWorkflowError("ErrResourceConflict", "ledger account has active children"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "ledger account has active children"),
		},
		"close a category that does not exist": {
			Method: http.MethodPost, Path: "/api/categories/99/close",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CloseCategory,
				Params:   ct.CloseCategoryInput{LedgerAccountID: 99},
				Err:      temporal.NewApplicationError("resource LedgerAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 99"),
		},
	})
}
