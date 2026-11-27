package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
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
				Params:   ct.CloseCategoryInput{CategoryID: 11},
			},
			WantStatus: http.StatusNoContent,
		},
		"close a category with open categories under it": {
			Method: http.MethodPost, Path: "/api/categories/10/close",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CloseCategory,
				Params:   ct.CloseCategoryInput{CategoryID: 10},
				Err:      temporal.NewApplicationError("resource Category conflict: category 10 has open categories under it; close them first", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Category conflict: category 10 has open categories under it; close them first"),
		},
		"close a category that does not exist": {
			Method: http.MethodPost, Path: "/api/categories/99/close",
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.CloseCategory,
				Params:   ct.CloseCategoryInput{CategoryID: 99},
				Err:      temporal.NewApplicationError("resource Category not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Category not found with ID: 99"),
		},
	})
}
