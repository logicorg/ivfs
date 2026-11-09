package counterparties

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDelete checks that the path becomes DeleteCounterparty's input, that the
// response says how many entries lost the counterparty, and that a conflict
// that outlasts the workflow's rounds keeps its status.
func TestDelete(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete a counterparty entries name": {
			Method: http.MethodDelete, Path: "/api/counterparties/3",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.DeleteCounterparty,
				Params:   lt.DeleteCounterpartyInput{CounterpartyID: 3},
				Result:   lt.DeleteCounterpartyResult{Cleared: 4},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"cleared":4}`,
		},
		"delete a counterparty that does not exist": {
			Method: http.MethodDelete, Path: "/api/counterparties/9",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.DeleteCounterparty,
				Params:   lt.DeleteCounterpartyInput{CounterpartyID: 9},
				Result:   lt.DeleteCounterpartyResult{},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"cleared":0}`,
		},
		"delete a counterparty entries keep being recorded with": {
			Method: http.MethodDelete, Path: "/api/counterparties/3",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.DeleteCounterparty,
				Params:   lt.DeleteCounterpartyInput{CounterpartyID: 3},
				Err:      temporal.NewApplicationError("resource Counterparty conflict: still referenced by ledger.journal_entry.counterparty_id", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Counterparty conflict: still referenced by ledger.journal_entry.counterparty_id"),
		},
		"delete a counterparty with an id that is not positive": {
			Method: http.MethodDelete, Path: "/api/counterparties/0",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
