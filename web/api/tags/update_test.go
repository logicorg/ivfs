package tags

import (
	"net/http"
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestUpdate checks that only the fields in the body reach UpdateTag, that
// the workflow's errors keep their status, and that what the input types
// forbid runs no workflow.
func TestUpdate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename a tag": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"name":"holiday"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 3, Name: ptr.NewPtr("holiday")},
			},
			WantStatus: http.StatusNoContent,
		},
		"describe a tag": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"description":"time off"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 3, Description: ptr.NewPtr("time off")},
			},
			WantStatus: http.StatusNoContent,
		},
		"clear a tag's description": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"description":""}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 3, Description: ptr.NewPtr("")},
			},
			WantStatus: http.StatusNoContent,
		},
		"rename a tag that does not exist": {
			Method: http.MethodPatch, Path: "/api/tags/9", Body: `{"name":"holiday"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 9, Name: ptr.NewPtr("holiday")},
				Err:      temporal.NewApplicationError("resource Tag not found with ID: 9", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Tag not found with ID: 9"),
		},
		"rename a tag to a name another has": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"name":"holiday"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 3, Name: ptr.NewPtr("holiday")},
				Err:      temporal.NewApplicationError("resource Tag conflict: already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Tag conflict: already exists"),
		},
		"rename a tag to a blank name": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"name":"   "}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.UpdateTag,
				Params:   &tt.UpdateTagParams{ID: 3, Name: ptr.NewPtr("   ")},
				Err:      temporal.NewApplicationError("name must not be blank", "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "name must not be blank"),
		},
		"rename a tag to no name": {
			Method: http.MethodPatch, Path: "/api/tags/3", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"update a tag with an invalid id": {
			Method: http.MethodPatch, Path: "/api/tags/0", Body: `{"name":"holiday"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
