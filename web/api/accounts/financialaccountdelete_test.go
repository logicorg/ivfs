package accounts

import (
	"net/http"
	"testing"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestDeleteFinancialAccount checks that deleting runs DeleteFinancialAccount
// under the account's deletion ID, with the replacement if one is asked for,
// and that its refusals keep their statuses.
func TestDeleteFinancialAccount(t *testing.T) {
	remove := func(params lt.DeleteFinancialAccountInput, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: ledgeraccountswf.DeleteFinancialAccount, Params: params, Result: lt.DeleteAccountResult{}, Err: err,
			WorkflowID: workflows.DeleteFinancialAccountWorkflowID(params.FinancialAccountID),
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete it": {
			Method: http.MethodDelete, Path: "/api/financial-accounts/3",
			Call:       remove(lt.DeleteFinancialAccountInput{FinancialAccountID: 3}, nil),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"merge it into a replacement": {
			Method: http.MethodDelete, Path: "/api/financial-accounts/3?replacementId=4",
			Call: remove(lt.DeleteFinancialAccountInput{FinancialAccountID: 3, ReplacementID: ptr.Int64(4)},
				temporal.NewApplicationError("merging an account into a replacement is not supported yet; delete it without a replacement", "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"merging an account into a replacement is not supported yet; delete it without a replacement"),
		},
		"a ledger account with a child outside it": {
			Method: http.MethodDelete, Path: "/api/financial-accounts/3",
			Call: remove(lt.DeleteFinancialAccountInput{FinancialAccountID: 3},
				temporal.NewApplicationError("resource LedgerAccount conflict: ledger account 9 has child accounts outside it; delete or move them first", "ErrResourceConflict")),
			WantStatus: http.StatusConflict,
			WantBody: apitest.Problem(http.StatusConflict,
				"resource LedgerAccount conflict: ledger account 9 has child accounts outside it; delete or move them first"),
		},
		"id 0": {
			Method: http.MethodDelete, Path: "/api/financial-accounts/0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
