package accounts

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestDeleteLedgerAccount checks that deleting runs DeleteLedgerAccount under
// the account's deletion ID, with the replacement if one is asked for, and
// returns its warnings and its refusals' statuses.
func TestDeleteLedgerAccount(t *testing.T) {
	remove := func(params lt.DeleteLedgerAccountInput, result lt.DeleteAccountResult, err error) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{
			Workflow: ledgeraccountswf.DeleteLedgerAccount, Params: params, Result: result, Err: err,
			WorkflowID: workflows.DeleteLedgerAccountWorkflowID(params.LedgerAccountID),
		}
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"delete it": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/7",
			Call:       remove(lt.DeleteLedgerAccountInput{LedgerAccountID: 7}, lt.DeleteAccountResult{}, nil),
			WantStatus: http.StatusOK, WantBody: `{}`,
		},
		"delete it leaving a holding negative": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/7",
			Call: remove(lt.DeleteLedgerAccountInput{LedgerAccountID: 7}, lt.DeleteAccountResult{Warnings: []lt.NegativeHolding{{
				FinancialAccountID: 3, FinancialAccountName: "IBKR", AssetID: 12, AssetName: "ACME",
				At: time.Date(2026, 3, 3, 0, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-10),
			}}}, nil),
			WantStatus: http.StatusOK,
			WantBody: `{"warnings":[{"financialAccountId":3,"financialAccountName":"IBKR",` +
				`"assetId":12,"assetName":"ACME","at":"2026-03-03T00:00:00Z","units":"-10"}]}`,
		},
		"merge it into a replacement": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/7?replacementId=8",
			Call: remove(lt.DeleteLedgerAccountInput{LedgerAccountID: 7, ReplacementID: ptr.Int64(8)}, lt.DeleteAccountResult{},
				temporal.NewApplicationError("merging an account into a replacement is not supported yet; delete it without a replacement", "ErrInvalidParameter")),
			WantStatus: http.StatusUnprocessableEntity,
			WantBody: apitest.Problem(http.StatusUnprocessableEntity,
				"merging an account into a replacement is not supported yet; delete it without a replacement"),
		},
		"a ledger account with children": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/7",
			Call: remove(lt.DeleteLedgerAccountInput{LedgerAccountID: 7}, lt.DeleteAccountResult{},
				temporal.NewApplicationError("resource LedgerAccount conflict: ledger account 7 has child accounts; delete or move them first", "ErrResourceConflict")),
			WantStatus: http.StatusConflict,
			WantBody: apitest.Problem(http.StatusConflict,
				"resource LedgerAccount conflict: ledger account 7 has child accounts; delete or move them first"),
		},
		"id 0": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/0", WantStatus: http.StatusUnprocessableEntity,
		},
		"replacement 0": {
			Method: http.MethodDelete, Path: "/api/ledger-accounts/7?replacementId=0", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
