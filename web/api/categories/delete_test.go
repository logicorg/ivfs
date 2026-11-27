package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDelete checks that deleting runs DeleteCategory, with the replacement
// if one is asked for, and that its refusals keep their statuses.
func TestDelete(t *testing.T) {
	remove := func(params ct.DeleteCategoryInput, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: categorieswf.DeleteCategory, Params: params, Err: err}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete it": {
			Method: http.MethodDelete, Path: "/api/categories/11",
			Call:       remove(ct.DeleteCategoryInput{CategoryID: 11}, nil),
			WantStatus: http.StatusNoContent,
		},
		"merge it into a replacement": {
			Method: http.MethodDelete, Path: "/api/categories/11?replacementId=12",
			Call:       remove(ct.DeleteCategoryInput{CategoryID: 11, ReplacementID: ptr.Int64(12)}, nil),
			WantStatus: http.StatusNoContent,
		},
		"merge it into a closed replacement": {
			Method: http.MethodDelete, Path: "/api/categories/11?replacementId=12",
			Call: remove(ct.DeleteCategoryInput{CategoryID: 11, ReplacementID: ptr.Int64(12)},
				temporal.NewApplicationError("replacement category 12 is closed; choose an open one", "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "replacement category 12 is closed; choose an open one"),
		},
		"a category with categories under it": {
			Method: http.MethodDelete, Path: "/api/categories/10",
			Call: remove(ct.DeleteCategoryInput{CategoryID: 10}, temporal.NewApplicationError(
				"resource Category conflict: category 10 has categories under it; delete or move them first", "ErrResourceConflict")),
			WantStatus: http.StatusConflict,
			WantBody: apitest.Problem(http.StatusConflict,
				"resource Category conflict: category 10 has categories under it; delete or move them first"),
		},
		"id 0": {
			Method: http.MethodDelete, Path: "/api/categories/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
