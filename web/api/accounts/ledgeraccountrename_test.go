package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestRenameLedgerAccount checks that the path and body become
// RenameLedgerAccount's input, and that its refusals keep their statuses.
func TestRenameLedgerAccount(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"rename it": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/rename", Body: `{"name":"Savings vault"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameLedgerAccount,
				Params:   lt.RenameLedgerAccountInput{LedgerAccountID: 7, Name: "Savings vault"},
				Result:   lt.LedgerAccountMutationResult{LedgerAccountID: 7},
			},
			WantStatus: http.StatusNoContent,
		},
		"a name another ledger account has": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/rename", Body: `{"name":"Cash"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameLedgerAccount,
				Params:   lt.RenameLedgerAccountInput{LedgerAccountID: 7, Name: "Cash"},
				Err:      temporal.NewApplicationError("a LedgerAccount with this name already exists", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "a LedgerAccount with this name already exists"),
		},
		"an unknown id": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/99/rename", Body: `{"name":"Cash"}`,
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.RenameLedgerAccount,
				Params:   lt.RenameLedgerAccountInput{LedgerAccountID: 99, Name: "Cash"},
				Err:      temporal.NewApplicationError("resource LedgerAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 99"),
		},
		"an empty name": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/7/rename", Body: `{"name":""}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"id 0": {
			Method: http.MethodPost, Path: "/api/ledger-accounts/0/rename", Body: `{"name":"Cash"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
