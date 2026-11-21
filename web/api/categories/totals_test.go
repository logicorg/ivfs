package categories

import (
	"net/http"
	"testing"
	"time"

	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
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
	month := rpt.CategoryTotalsParams{From: january, To: february}
	path := "/api/categories/totals?from=2026-01-01T00:00:00Z&to=2026-02-01T00:00:00Z"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"total a month": {
			Method: http.MethodGet, Path: path,
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.CategoryTotals, Params: month,
				Result: rpt.CategoryTotalsResult{Totals: []rpt.CategoryTotal{
					{CategoryID: ptr.Int64(10), Kind: model.LedgerAccountIncome, CurrencyID: 1, CurrencyCode: "EUR", Total: decimal.RequireFromString("2000")},
					{CategoryID: ptr.Int64(11), Kind: model.LedgerAccountExpense, CurrencyID: 1, CurrencyCode: "EUR", Total: decimal.RequireFromString("-20.5")},
					{Kind: model.LedgerAccountExpense, CurrencyID: 2, CurrencyCode: "USD", Total: decimal.RequireFromString("15")},
				}},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"totals":[` +
				`{"categoryId":10,"kind":"income","currencyId":1,"currencyCode":"EUR","total":"2000"},` +
				`{"categoryId":11,"kind":"expense","currencyId":1,"currencyCode":"EUR","total":"-20.5"},` +
				`{"kind":"expense","currencyId":2,"currencyCode":"USD","total":"15"}]}`,
		},
		"total a month without entries": {
			Method: http.MethodGet, Path: path,
			Call:       &apitest.WorkflowCall{Workflow: reportingwf.CategoryTotals, Params: month, Result: rpt.CategoryTotalsResult{}},
			WantStatus: http.StatusOK, WantBody: `{"totals":[]}`,
		},
		"total a period that ends before it starts": {
			Method: http.MethodGet, Path: "/api/categories/totals?from=2026-02-01T00:00:00Z&to=2026-01-01T00:00:00Z",
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.CategoryTotals, Params: rpt.CategoryTotalsParams{From: february, To: january},
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
