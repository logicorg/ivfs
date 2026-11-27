package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestRename checks that the path and body become RenameCategory's input,
// and that a conflict is answered with the activity's message.
func TestRename(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename a category": {
			Method: http.MethodPost, Path: "/api/categories/11/rename", Body: `{"name":"Supermarket"}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.RenameCategory,
				Params:   ct.RenameCategoryInput{CategoryID: 11, Name: "Supermarket"},
				Result:   ct.CategoryResult{CategoryID: 11},
			},
			WantStatus: http.StatusNoContent,
		},
		"rename a category to a name its sibling has": {
			Method: http.MethodPost, Path: "/api/categories/11/rename", Body: `{"name":"Food"}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.RenameCategory,
				Params:   ct.RenameCategoryInput{CategoryID: 11, Name: "Food"},
				Err:      temporal.NewApplicationError("a Category with this name already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "a Category with this name already exists"),
		},
	})
}
