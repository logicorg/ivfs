package tags

import (
	"net/http"
	"testing"

	tt "github.com/logicorg/ivfs/finance/common/types/tagtypes"
	tagswf "github.com/logicorg/ivfs/finance/workflows/tags"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestList checks that each tag becomes its projection and that no tags are
// an empty list.
func TestList(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"every tag": {
			Method: http.MethodGet, Path: "/api/tags",
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.ListTags,
				Params:   &tt.ListTagsParams{},
				Result: tt.ListTagsResult{Tags: []*model.Tag{
					{ID: 1, Name: "reimbursable", Description: "paid back by work"},
					{ID: 2, Name: "vacation"},
				}},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"tags":[{"id":1,"name":"reimbursable","description":"paid back by work"},{"id":2,"name":"vacation","description":""}]}`,
		},
		"no tags": {
			Method: http.MethodGet, Path: "/api/tags",
			Call: &apitest.WorkflowCall{
				Workflow: tagswf.ListTags,
				Params:   &tt.ListTagsParams{},
				Result:   tt.ListTagsResult{},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"tags":[]}`,
		},
	})
}
