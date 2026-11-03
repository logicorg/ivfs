package accounts

import (
	"net/http"
	"testing"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestDeleteInstitution checks that deleting runs DeleteInstitution under the
// institution's deletion ID, and that its refusals keep their statuses.
func TestDeleteInstitution(t *testing.T) {
	remove := func(err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: institutionwf.DeleteInstitution, Params: it.DeleteInstitutionParams{FindBy: it.RetrieveInstitutionParams{ID: 5}},
			Err: err, WorkflowID: workflows.DeleteInstitutionWorkflowID(5),
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete it": {
			Method: http.MethodDelete, Path: "/api/institutions/5",
			Call:       remove(nil),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"an account that fails to delete": {
			Method: http.MethodDelete, Path: "/api/institutions/5",
			Call: remove(apitest.ChildWorkflowError("ErrResourceConflict",
				"resource LedgerAccount conflict: ledger account 9 has child accounts outside it; delete or move them first")),
			WantStatus: http.StatusConflict,
			WantBody: apitest.Problem(http.StatusConflict,
				"resource LedgerAccount conflict: ledger account 9 has child accounts outside it; delete or move them first"),
		},
		"id 0": {
			Method: http.MethodDelete, Path: "/api/institutions/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
