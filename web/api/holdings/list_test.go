package holdings

import (
	"net/http"
	"testing"
	"time"

	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/apitest"
	"github.com/shopspring/decimal"
)

// TestList checks that the filters and the instant reach ListPositionsAt,
// that each holding keeps its exact units and cost as strings with its
// average cost only when there is one, and that none is an empty list.
func TestList(t *testing.T) {
	list := func(params rpt.PositionsAtParams, positions ...rpt.PositionResult) *apitest.WorkflowCall {
		return &apitest.WorkflowCall{Workflow: reportingwf.ListPositionsAt, Params: params, Result: rpt.PositionsAtResult{Positions: positions}}
	}
	vwce := rpt.PositionResult{
		FinancialAccountID: 3, AssetID: 7, Units: decimal.RequireFromString("12.5"),
		CostBasis:   []rpt.CostBasisAmount{{CurrencyID: 1, Amount: decimal.RequireFromString("1250.25")}},
		AverageCost: &rpt.CostBasisAmount{CurrencyID: 1, Amount: decimal.RequireFromString("100.02")},
	}
	vwceBody := `{"financialAccountId":3,"assetId":7,"units":"12.5","costBasis":[{"currencyId":1,"amount":"1250.25"}],` +
		`"unknownCostBasis":false,"averageCost":{"currencyId":1,"amount":"100.02"}}`
	gift := rpt.PositionResult{
		FinancialAccountID: 3, AssetID: 8, Units: decimal.RequireFromString("4"),
		CostBasis: []rpt.CostBasisAmount{
			{CurrencyID: 1, Amount: decimal.RequireFromString("10")}, {CurrencyID: 2, Amount: decimal.RequireFromString("20")},
		},
		UnknownCostBasis: true,
	}
	giftBody := `{"financialAccountId":3,"assetId":8,"units":"4","costBasis":[{"currencyId":1,"amount":"10"},` +
		`{"currencyId":2,"amount":"20"}],"unknownCostBasis":true}`

	apitest.RunCases(t, Register, map[string]apitest.Case{
		"every holding now": {
			Method: http.MethodGet, Path: "/api/holdings",
			Call:       list(rpt.PositionsAtParams{}, vwce, gift),
			WantStatus: http.StatusOK, WantBody: `{"holdings":[` + vwceBody + `,` + giftBody + `]}`,
		},
		"an account's holding of an asset at an instant": {
			Method: http.MethodGet, Path: "/api/holdings?financialAccountId=3&assetId=7&at=2026-01-31T23:59:59Z",
			Call: list(rpt.PositionsAtParams{
				FinancialAccountID: 3, AssetID: 7, At: time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC),
			}, vwce),
			WantStatus: http.StatusOK, WantBody: `{"holdings":[` + vwceBody + `]}`,
		},
		"a holding sold to zero": {
			Method: http.MethodGet, Path: "/api/holdings",
			Call:       list(rpt.PositionsAtParams{}, rpt.PositionResult{FinancialAccountID: 3, AssetID: 9, Units: decimal.Zero}),
			WantStatus: http.StatusOK,
			WantBody:   `{"holdings":[{"financialAccountId":3,"assetId":9,"units":"0","costBasis":[],"unknownCostBasis":false}]}`,
		},
		"no holdings": {
			Method: http.MethodGet, Path: "/api/holdings",
			Call:       list(rpt.PositionsAtParams{}),
			WantStatus: http.StatusOK, WantBody: `{"holdings":[]}`,
		},
		"an account that is not one": {
			Method: http.MethodGet, Path: "/api/holdings?financialAccountId=0", WantStatus: http.StatusUnprocessableEntity,
		},
		"an instant that is not one": {
			Method: http.MethodGet, Path: "/api/holdings?at=yesterday", WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
