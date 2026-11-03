package categories

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

// TestTotals checks that the period reaches CategoryTotals, that each total
// is answered as an exact amount, and that a period the workflow rejects, or
// one not given, is a 422.
func TestTotals(t *testing.T) {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)
	month := lt.CategoryTotalsParams{From: january, To: february}
	path := "/api/categories/totals?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"total a month": {
			Method: http.MethodGet, Path: path,
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.CategoryTotals, Params: month,
				Result: lt.CategoryTotalsResult{Totals: []lt.CategoryTotal{
					{LedgerAccountID: 10, Kind: model.LedgerAccountIncome, CurrencyID: 1, CurrencyCode: "EUR", Total: decimal.RequireFromString("2000")},
					{LedgerAccountID: 11, Kind: model.LedgerAccountExpense, CurrencyID: 1, CurrencyCode: "EUR", Total: decimal.RequireFromString("-20.5")},
				}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"totals":[` +
				`{"id":10,"kind":"income","currencyId":1,"currencyCode":"EUR","total":"2000"},` +
				`{"id":11,"kind":"expense","currencyId":1,"currencyCode":"EUR","total":"-20.5"}]}`,
		},
		"total a month without entries": {
			Method: http.MethodGet, Path: path,
			Call:       &apitest.WorkflowCall{Workflow: reportingwf.CategoryTotals, Params: month, Result: lt.CategoryTotalsResult{}},
			WantStatus: http.StatusOK, WantBody: `{"totals":[]}`,
		},
		"total a period that ends before it starts": {
			Method: http.MethodGet, Path: "/api/categories/totals?from=2026-02-01T00:00:00Z&to=2026-01-01T00:00:00Z",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.CategoryTotals, Params: lt.CategoryTotalsParams{From: february, To: january},
				Err: temporal.NewApplicationError("invalid parameter: the end of the period must be after its start", "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
		},
		"total without an end": {
			Method: http.MethodGet, Path: "/api/categories/totals?from=2026-01-01T00:00:00Z",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
