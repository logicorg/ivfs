package networth

import (
	"net/http"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
)

// TestSeries checks that the days reach NetWorthSeries, that the rows are
// answered with their dates as days and their figures as exact amounts, and
// that a range that is not one is a 422.
func TestSeries(t *testing.T) {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	month := lt.NetWorthSeriesParams{From: january, To: january.AddDate(0, 0, 30)}
	path := "/api/net-worth/series?from=2026-01-01&to=2026-01-31"

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"a month": {
			Method: http.MethodGet, Path: path,
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.NetWorthSeries, Params: month,
				Result: lt.NetWorthSeriesResult{
					Balances: []lt.NetWorthBalance{{Date: january, CurrencyID: 1, Assets: decimal.RequireFromString("1000.25"), Liabilities: decimal.RequireFromString("50")}},
					Units:    []lt.NetWorthUnits{{Date: january.AddDate(0, 0, 9), AssetID: 3, Units: decimal.RequireFromString("-0.5")}},
				},
			},
			WantStatus: http.StatusOK,
			WantBody: `{"balances":[{"date":"2026-01-01","currencyId":1,"assets":"1000.25","liabilities":"50"}],` +
				`"units":[{"date":"2026-01-10","assetId":3,"units":"-0.5"}]}`,
		},
		"a month with nothing held": {
			Method: http.MethodGet, Path: path,
			Call:       &apitest.WorkflowCall{Workflow: reportingwf.NetWorthSeries, Params: month, Result: lt.NetWorthSeriesResult{}},
			WantStatus: http.StatusOK, WantBody: `{"balances":[],"units":[]}`,
		},
		"a range of too many changes": {
			Method: http.MethodGet, Path: path,
			Call: &apitest.WorkflowCall{
				Workflow: reportingwf.NetWorthSeries, Params: month,
				Err: temporal.NewApplicationError("invalid parameter: the range holds 10001 changes; shorten it", "ErrInvalidParameter"),
			},
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a range without an end": {
			Method: http.MethodGet, Path: "/api/net-worth/series?from=2026-01-01",
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a range with an instant for a day": {
			Method: http.MethodGet, Path: "/api/net-worth/series?from=2026-01-01T00:00:00Z&to=2026-01-31",
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
