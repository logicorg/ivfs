package journal

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestSetTags checks that the tags are passed to SetJournalEntryTags with the
// entry's id, that the tags the entry ends with are returned, as an empty list
// when it has none, that a refusal keeps its status, and that a request Huma
// rejects runs no workflow.
func TestSetTags(t *testing.T) {
	set := func(tagIDs []int64, result lt.SetJournalEntryTagsResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: journalwf.SetJournalEntryTags,
			Params:   lt.SetJournalEntryTagsInput{JournalEntryID: 7, TagIDs: tagIDs},
			Result:   result,
			Err:      err,
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"set the tags": {
			Method: http.MethodPut, Path: "/api/journal-entries/7/tags", Body: `{"tagIds":[4,3,4]}`,
			Call:       set([]int64{4, 3, 4}, lt.SetJournalEntryTagsResult{TagIDs: []int64{3, 4}}, nil),
			WantStatus: http.StatusOK, WantBody: `{"tagIds":[3,4]}`,
		},
		"clear the tags": {
			Method: http.MethodPut, Path: "/api/journal-entries/7/tags", Body: `{"tagIds":[]}`,
			Call:       set([]int64{}, lt.SetJournalEntryTagsResult{}, nil),
			WantStatus: http.StatusOK, WantBody: `{"tagIds":[]}`,
		},
		"an unknown tag": {
			Method: http.MethodPut, Path: "/api/journal-entries/7/tags", Body: `{"tagIds":[99]}`,
			Call: set([]int64{99}, lt.SetJournalEntryTagsResult{},
				apitest.ChildWorkflowError("ErrNotFound", "resource Tag not found")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Tag not found"),
		},
		"an unknown entry": {
			Method: http.MethodPut, Path: "/api/journal-entries/7/tags", Body: `{"tagIds":[3]}`,
			Call: set([]int64{3}, lt.SetJournalEntryTagsResult{},
				temporal.NewApplicationError("resource JournalEntry not found with ID: 7", "ErrNotFound")),
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource JournalEntry not found with ID: 7"),
		},
		"no tags field": {
			Method: http.MethodPut, Path: "/api/journal-entries/7/tags", Body: `{}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an id that is not one": {
			Method: http.MethodPut, Path: "/api/journal-entries/0/tags", Body: `{"tagIds":[3]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
