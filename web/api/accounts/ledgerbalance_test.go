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

// TestLedgerBalance checks that the balance keeps its exact amount, that the
// instant reaches LedgerAccountBalance, now when omitted, and that a ledger
// account that does not exist is a 404.
func TestLedgerBalance(t *testing.T) {
	card := lt.LedgerAccountBalanceResult{
		LedgerAccountID: 101, LedgerAccountName: "Card balance", Kind: model.LedgerAccountLiability,
		CurrencyID: 2, CurrencyCode: "USD", Balance: decimal.RequireFromString("-0.01"),
	}
	cardBody := `{"ledgerAccountId":101,"ledgerAccountName":"Card balance","kind":"liability","currencyId":2,"currencyCode":"USD","balance":"-0.01","closed":false}`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"the balance now": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/101/balance",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.LedgerAccountBalance,
				Params:   lt.LedgerAccountBalanceParams{LedgerAccountID: 101},
				Result:   card,
			},
			WantStatus: http.StatusOK, WantBody: cardBody,
		},
		"the balance at an instant": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/101/balance?at=2026-01-31T23:59:59Z",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.LedgerAccountBalance,
				Params: lt.LedgerAccountBalanceParams{
					LedgerAccountID: 101, At: time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
				},
				Result: card,
			},
			WantStatus: http.StatusOK, WantBody: cardBody,
		},
		"a ledger account that does not exist": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/99/balance",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.LedgerAccountBalance,
				Params:   lt.LedgerAccountBalanceParams{LedgerAccountID: 99},
				Err:      temporal.NewApplicationError("resource LedgerAccount not found with ID: 99", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource LedgerAccount not found with ID: 99"),
		},
		"an id that is not one": {
			Method: http.MethodGet, Path: "/api/ledger-accounts/0/balance", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
