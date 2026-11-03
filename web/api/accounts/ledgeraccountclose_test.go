package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCloseLedgerAccount checks that closing runs CloseLedgerAccount with no closing date,
// so the activity closes it now, and that its refusals keep their statuses.
func TestCloseLedgerAccount(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"close it": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/close",
			Call:       &apitest.WorkflowCall{Workflow: ledgeraccountswf.CloseLedgerAccount, Params: lt.CloseLedgerAccountInput{LedgerAccountID: 7}},
			WantStatus: http.StatusNoContent,
		},
		"a ledger account with a balance": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/close",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CloseLedgerAccount, Params: lt.CloseLedgerAccountInput{LedgerAccountID: 7},
				Err: temporal.NewApplicationError("resource LedgerAccount conflict: ledger account still has a balance of 100", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount conflict: ledger account still has a balance of 100"),
		},
		"a ledger account with children": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/close",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CloseLedgerAccount, Params: lt.CloseLedgerAccountInput{LedgerAccountID: 7},
				Err: temporal.NewApplicationError("resource LedgerAccount conflict: ledger account has children", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource LedgerAccount conflict: ledger account has children"),
		},
		"an unknown id": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/99/close",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CloseLedgerAccount, Params: lt.CloseLedgerAccountInput{LedgerAccountID: 99},
				Err: temporal.NewApplicationError("resource LedgerAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 99"),
		},
		"id 0": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/0/close",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
