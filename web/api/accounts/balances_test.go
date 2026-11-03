package accounts

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestBalances checks that each balance keeps its exact amount as a string,
// that the instant reaches FinancialAccountBalances, now when omitted, and
// that an account that does not exist is a 404.
func TestBalances(t *testing.T) {
	balances := lt.FinancialAccountBalancesResult{FinancialAccountID: 10, Balances: []lt.LedgerAccountBalanceResult{
		{
			LedgerAccountID: 100, LedgerAccountName: "Cash", Kind: model.LedgerAccountAsset,
			CurrencyID: 1, CurrencyCode: "EUR", Balance: decimal.RequireFromString("1200.50"),
		},
		{
			LedgerAccountID: 101, LedgerAccountName: "Card balance", Kind: model.LedgerAccountLiability,
			CurrencyID: 2, CurrencyCode: "USD", Balance: decimal.RequireFromString("-0.01"),
		},
	}}
	wantBalances := `{"financialAccountId":10,"balances":[` +
		`{"ledgerAccountId":100,"ledgerAccountName":"Cash","kind":"asset","currencyId":1,"currencyCode":"EUR","balance":"1200.5","closed":false},` +
		`{"ledgerAccountId":101,"ledgerAccountName":"Card balance","kind":"liability","currencyId":2,"currencyCode":"USD","balance":"-0.01","closed":false}]}`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"the balances now": {
			Method: http.MethodGet, Path: "/api/financial-accounts/10/balances",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.FinancialAccountBalances,
				Params:   lt.FinancialAccountBalancesParams{FinancialAccountID: 10},
				Result:   balances,
			},
			WantStatus: http.StatusOK, WantBody: wantBalances,
		},
		"the balances at an instant": {
			Method: http.MethodGet, Path: "/api/financial-accounts/10/balances?at=2026-01-31T23:59:59Z",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.FinancialAccountBalances,
				Params: lt.FinancialAccountBalancesParams{
					FinancialAccountID: 10, At: time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
				},
				Result: balances,
			},
			WantStatus: http.StatusOK, WantBody: wantBalances,
		},
		"an account with no balances": {
			Method: http.MethodGet, Path: "/api/financial-accounts/10/balances",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.FinancialAccountBalances,
				Params:   lt.FinancialAccountBalancesParams{FinancialAccountID: 10},
				Result:   lt.FinancialAccountBalancesResult{FinancialAccountID: 10},
			},
			WantStatus: http.StatusOK, WantBody: `{"financialAccountId":10,"balances":[]}`,
		},
		"an account that does not exist": {
			Method: http.MethodGet, Path: "/api/financial-accounts/99/balances",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.FinancialAccountBalances,
				Params:   lt.FinancialAccountBalancesParams{FinancialAccountID: 99},
				Err:      temporal.NewApplicationError("resource FinancialAccount 99 not found", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource FinancialAccount 99 not found"),
		},
		"an instant that is not one": {
			Method: http.MethodGet, Path: "/api/financial-accounts/10/balances?at=yesterday",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
