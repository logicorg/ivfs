package tags

import (
	"net/http"
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestDelete checks that the id is passed to DeleteTag and that an invalid id
// runs no workflow, since an id of 0 would make the activity match by name.
func TestDelete(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete a tag": {
			Method: http.MethodDelete, Path: "/api/tags/3",
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.DeleteTag,
				Params:   &tt.DeleteTagParams{ID: 3},
			},
			WantStatus: http.StatusNoContent,
		},
		"delete a tag with an invalid id": {
			Method: http.MethodDelete, Path: "/api/tags/0",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
