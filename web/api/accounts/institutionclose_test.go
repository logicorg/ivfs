package accounts

import (
	"net/http"
	"testing"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	institutionwf "github.com/logicorg/ivfs/finance/workflows/institution"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCloseInstitution checks that closing runs CloseInstitution with no closing date, so the
// activity closes it now, and that its refusals keep their statuses.
func TestCloseInstitution(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"close it": {
			Method: http.MethodPost, Path: "/api/institutions/7/close",
			Call:       &apitest.WorkflowCall{Workflow: institutionwf.CloseInstitution, Params: it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: 7}}},
			WantStatus: http.StatusNoContent,
		},
		"an institution with open accounts": {
			Method: http.MethodPost, Path: "/api/institutions/7/close",
			Call: &apitest.WorkflowCall{
				Workflow: institutionwf.CloseInstitution, Params: it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: 7}},
				Err: temporal.NewApplicationError("resource Institution conflict: institution has open financial accounts", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Institution conflict: institution has open financial accounts"),
		},
		"an unknown id": {
			Method: http.MethodPost, Path: "/api/institutions/99/close",
			Call: &apitest.WorkflowCall{
				Workflow: institutionwf.CloseInstitution, Params: it.CloseParams{FindBy: it.RetrieveInstitutionParams{ID: 99}},
				Err: temporal.NewApplicationError("resource Institution not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Institution not found with ID: 99"),
		},
		"id 0": {
			Method: http.MethodPost, Path: "/api/institutions/0/close",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
