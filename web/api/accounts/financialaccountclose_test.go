package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestCloseFinancialAccount checks that closing runs CloseFinancialAccount with no closing date, so the
// activity closes it now, and that its refusals keep their statuses.
func TestCloseFinancialAccount(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"close it": {
			Method: http.MethodPost, Path: "/api/financial-accounts/7/close",
			Call:       &apitest.WorkflowCall{Workflow: ledgeraccountswf.CloseFinancialAccount, Params: lt.CloseFinancialAccountInput{FinancialAccountID: 7}},
			WantStatus: http.StatusNoContent,
		},
		"a financial account with open ledger accounts": {
			Method: http.MethodPost, Path: "/api/financial-accounts/7/close",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CloseFinancialAccount, Params: lt.CloseFinancialAccountInput{FinancialAccountID: 7},
				Err: temporal.NewApplicationError("resource FinancialAccount conflict: financial account has active linked ledger accounts", "ErrResourceConflict"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource FinancialAccount conflict: financial account has active linked ledger accounts"),
		},
		"an unknown id": {
			Method: http.MethodPost, Path: "/api/financial-accounts/99/close",
			Call: &apitest.WorkflowCall{
				Workflow: ledgeraccountswf.CloseFinancialAccount, Params: lt.CloseFinancialAccountInput{FinancialAccountID: 99},
				Err: temporal.NewApplicationError("resource FinancialAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource FinancialAccount not found with ID: 99"),
		},
		"id 0": {
			Method: http.MethodPost, Path: "/api/financial-accounts/0/close",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
