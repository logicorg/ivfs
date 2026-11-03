package counterparties

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestRename checks that the path and body become RenameCounterparty's
// input, and that a missing counterparty or a name another one has keeps its
// status.
func TestRename(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename a counterparty": {
			Method: http.MethodPost, Path: "/api/counterparties/3/rename", Body: `{"name":"Tesco Express"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameCounterparty,
				Params:   lt.RenameCounterpartyInput{CounterpartyID: 3, Name: "Tesco Express"},
				Result:   lt.CounterpartyMutationResult{CounterpartyID: 3},
			},
			WantStatus: http.StatusNoContent,
		},
		"rename a counterparty to a name another has": {
			Method: http.MethodPost, Path: "/api/counterparties/3/rename", Body: `{"name":"Acme"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameCounterparty,
				Params:   lt.RenameCounterpartyInput{CounterpartyID: 3, Name: "Acme"},
				Err:      temporal.NewApplicationError("a Counterparty with this name already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "a Counterparty with this name already exists"),
		},
		"rename a counterparty that does not exist": {
			Method: http.MethodPost, Path: "/api/counterparties/9/rename", Body: `{"name":"Acme"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameCounterparty,
				Params:   lt.RenameCounterpartyInput{CounterpartyID: 9, Name: "Acme"},
				Err:      temporal.NewApplicationError("resource Counterparty not found with ID: 9", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Counterparty not found with ID: 9"),
		},
		"rename a counterparty to no name": {
			Method: http.MethodPost, Path: "/api/counterparties/3/rename", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
