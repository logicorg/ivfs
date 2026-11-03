package accounts

import (
	"net/http"
	"testing"
	"time"

	ast "github.com/logicorg/ivfs/finance/common/types/accountsetuptypes"
	"github.com/logicorg/ivfs/finance/workflows/facade/accountsetup"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestOpen checks that the form is passed to OpenAccount with its amounts
// exact (the data converter drops a trailing zero, so 1200.50 arrives as
// 1200.5), that both outcomes are answered, that a refusal keeps its status, and
// that a form Huma rejects runs no workflow.
func TestOpen(t *testing.T) {
	revolut := ast.OpenAccountInput{
		Type: ast.AccountTypeBank, InstitutionName: "Revolut",
		Currencies: []ast.CurrencyBalance{{Code: "EUR", OpeningBalance: decimal.RequireFromString("1200.5")}},
	}
	confirmed := revolut
	confirmed.CreateInstitution = true

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"open an account": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body: `{"type":"bank","institutionName":"Revolut","createInstitution":true,"currencies":[{"code":"EUR","openingBalance":"1200.50"}]}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.OpenAccount, Params: confirmed,
				Result: ast.OpenAccountResult{Outcome: ast.OutcomeOpened, InstitutionID: 2, InstitutionName: "Revolut", FinancialAccountID: 5, FinancialAccountName: "Revolut"},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"outcome":"opened","institutionId":2,"institutionName":"Revolut","financialAccountId":5,"financialAccountName":"Revolut"}`,
		},
		"an unknown institution, not confirmed": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body: `{"type":"bank","institutionName":"Revolut","currencies":[{"code":"EUR","openingBalance":"1200.50"}]}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.OpenAccount, Params: revolut,
				Result: ast.OpenAccountResult{Outcome: ast.OutcomeInstitutionNotFound, InstitutionName: "Revolut"},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"outcome":"institution_not_found","institutionName":"Revolut"}`,
		},
		"a wallet, from a date, with no opening balance": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body: `{"type":"cash","currencies":[{"code":"EUR"}],"openingBalancesAt":"2026-01-01T00:00:00Z"}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.OpenAccount,
				Params: ast.OpenAccountInput{
					Type: ast.AccountTypeCash, Currencies: []ast.CurrencyBalance{{Code: "EUR"}},
					OpeningBalancesAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
				},
				Result: ast.OpenAccountResult{Outcome: ast.OutcomeOpened, InstitutionID: 1, InstitutionName: "Physical cash", FinancialAccountID: 6, FinancialAccountName: "Wallet"},
			},
			WantStatus: http.StatusOK,
			WantBody:   `{"outcome":"opened","institutionId":1,"institutionName":"Physical cash","financialAccountId":6,"financialAccountName":"Wallet"}`,
		},
		"a closed institution": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body: `{"type":"bank","institutionName":"Revolut","currencies":[{"code":"EUR","openingBalance":"1200.50"}]}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.OpenAccount, Params: revolut,
				Err: temporal.NewApplicationError("resource Institution is closed", "ErrResourceClosed"),
			},
			WantStatus: http.StatusConflict,
			WantBody:   apitest.Problem(http.StatusConflict, "resource Institution is closed"),
		},
		"a bank account without an institution": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body: `{"type":"bank","currencies":[{"code":"EUR"}]}`,
			Call: &apitest.WorkflowCall{
				Workflow: accountsetup.OpenAccount,
				Params:   ast.OpenAccountInput{Type: ast.AccountTypeBank, Currencies: []ast.CurrencyBalance{{Code: "EUR"}}},
				Err:      temporal.NewApplicationError("the institution is required for a bank account", "ValidationError"),
			},
			WantStatus: http.StatusUnprocessableEntity,
			WantBody:   apitest.Problem(http.StatusUnprocessableEntity, "the institution is required for a bank account"),
		},
		"an unknown type": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body:       `{"type":"loan","institutionName":"Revolut","currencies":[{"code":"EUR"}]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no currencies": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body:       `{"type":"bank","institutionName":"Revolut","currencies":[]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an amount that is not a decimal": {
			Method: http.MethodPost, Path: "/api/accounts/open",
			Body:       `{"type":"bank","institutionName":"Revolut","currencies":[{"code":"EUR","openingBalance":"1,200"}]}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
