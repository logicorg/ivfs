package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDeletionPreview checks that the preview runs PreviewCategoryDeletion
// and returns its count of entries.
func TestDeletionPreview(t *testing.T) {
	preview := func(result ct.CategoryDeletionPreview, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: categorieswf.PreviewCategoryDeletion, Params: ct.PreviewCategoryDeletionInput{CategoryID: 11},
			Result: result, Err: err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"a category with entries": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call:       preview(ct.CategoryDeletionPreview{Entries: 4}, nil),
			WantStatus: http.StatusOK, WantBody: `{"entries":4}`,
		},
		"a category without entries": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call:       preview(ct.CategoryDeletionPreview{}, nil),
			WantStatus: http.StatusOK, WantBody: `{"entries":0}`,
		},
		"a category that does not exist": {
			Method: http.MethodGet, Path: "/api/categories/11/deletion-preview",
			Call:       preview(ct.CategoryDeletionPreview{}, temporal.NewApplicationError("resource Category not found with ID: 11", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Category not found with ID: 11"),
		},
		"id 0": {
			Method: http.MethodGet, Path: "/api/categories/0/deletion-preview", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
