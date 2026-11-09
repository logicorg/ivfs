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

// TestCreate checks that the body becomes InsertTag's params, with the
// description only when it is sent, and that a name the workflow refuses
// keeps its status.
func TestCreate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"create a tag": {
			Method: http.MethodPost, Path: "/api/tags", Body: `{"name":"vacation"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.InsertTag,
				Params:   tt.InsertTagParams{Name: "vacation"},
				Result:   tt.InsertTagResult{ID: 3},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"id":3}`,
		},
		"create a tag with a description": {
			Method: http.MethodPost, Path: "/api/tags", Body: `{"name":"vacation","description":"time off"}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.InsertTag,
				Params:   tt.InsertTagParams{Name: "vacation", Description: ptr.NewPtr("time off")},
				Result:   tt.InsertTagResult{ID: 3},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"id":3}`,
		},
		"create a tag with a blank name": {
			Method: http.MethodPost, Path: "/api/tags", Body: `{"name":"   "}`,
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.InsertTag,
				Params:   tt.InsertTagParams{Name: "   "},
				Err:      temporal.NewApplicationError("missing required parameters: Name", "ErrMissingRequiredParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "missing required parameters: Name"),
		},
		"create a tag with no name": {
			Method: http.MethodPost, Path: "/api/tags", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
