package accounts

import (
	"net/http"
	"testing"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows/facade/accountsetup"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestAddBalance checks that the form is passed to AddBalance for the account
// in the path, with its amount exact (the data converter drops a trailing
// zero, so 300.50 arrives as 300.5), that the added balance is picked out of
// the shared accounts the workflow also returns, that a refusal keeps its
// status, and that a form Huma rejects runs no workflow.
func TestAddBalance(t *testing.T) {
	vault := ast.AddBalanceInput{
		FinancialAccountID: 5, Name: "Savings vault", CurrencyCode: "EUR", OpeningBalance: decimal.RequireFromString("300.5"),
	}
	added := ast.AddBalanceResult{
		Outcome: ast.OutcomeOpened, LedgerAccountID: 12,
		Balances: ast.Balances{Balances: []lt.AccountSummary{
			{LedgerAccountID: 12, LedgerAccountName: "Savings vault"},
			{LedgerAccountID: 3, LedgerAccountName: "Opening balances"},
		}},
	}

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"add a balance": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body:       `{"name":"Savings vault","currencyCode":"EUR","openingBalance":"300.50"}`,
			Call:       &apitest.WorkflowCall{Workflow: accountsetup.AddBalance, Params: vault, Result: added},
			WantStatus: http.StatusOK,
			WantBody:   `{"ledgerAccountId":12,"ledgerAccountName":"Savings vault"}`,
		},
		"from a date, with no opening balance": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body: `{"name":"Savings vault","currencyCode":"EUR","openingBalanceAt":"2026-01-01T00:00:00Z"}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.AddBalance,
				Params: ast.AddBalanceInput{
					FinancialAccountID: 5, Name: "Savings vault", CurrencyCode: "EUR",
					OpeningBalanceAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Result: added,
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"ledgerAccountId":12,"ledgerAccountName":"Savings vault"}`,
		},
		"a closed account": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body: `{"name":"Savings vault","currencyCode":"EUR","openingBalance":"300.50"}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.AddBalance, Params: vault,
				Err: temporal.NewApplicationError("resource FinancialAccount is closed", "ErrResourceClosed"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource FinancialAccount is closed"),
		},
		"an unknown account": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body: `{"name":"Savings vault","currencyCode":"EUR","openingBalance":"300.50"}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.AddBalance, Params: vault,
				Err: temporal.NewApplicationError("resource FinancialAccount not found with ID: 5", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource FinancialAccount not found with ID: 5"),
		},
		"no name": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body:       `{"name":"","currencyCode":"EUR"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an amount that is not a decimal": {
			Method: http.MethodPost, Path: "/api/financial-accounts/5/balances",
			Body:       `{"name":"Savings vault","currencyCode":"EUR","openingBalance":"1,200"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"id 0": {
			Method: http.MethodPost, Path: "/api/financial-accounts/0/balances",
			Body:       `{"name":"Savings vault","currencyCode":"EUR"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
