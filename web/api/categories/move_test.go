package categories

import (
	"net/http"
	"testing"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/facade/categories"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestMove checks that a parent in the body moves the category under it, and
// that none moves it to the top level.
func TestMove(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"move a category under another": {
			Method: http.MethodPost, Path: "/api/categories/11/move", Body: `{"parentId":10}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.MoveCategory,
				Params:   ct.MoveCategoryInput{LedgerAccountID: 11, ParentID: ptr.Int64(10)},
				Result:   ct.CategoryResult{LedgerAccountID: 11},
			},
			WantStatus: http.StatusNoContent,
		},
		"move a category to the top level": {
			Method: http.MethodPost, Path: "/api/categories/11/move", Body: `{}`,
			Call: &apitest.WorkflowCall{
				Workflow: categorieswf.MoveCategory,
				Params:   ct.MoveCategoryInput{LedgerAccountID: 11},
				Result:   ct.CategoryResult{LedgerAccountID: 11},
			},
			WantStatus: http.StatusNoContent,
		},
	})
}
