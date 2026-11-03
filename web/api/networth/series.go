package networth

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	reportingwf "github.com/logicorg/ivfs/finance/workflows/ledger/reporting"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerSeries(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "get-net-worth-series",
		Method:      http.MethodGet,
		Path:        "/api/net-worth/series",
		Summary:     "List the balances and the units behind net worth over a range of days",
		Description: "Days are UTC days. A day counts every posted entry, and every movement in effect, that " +
			"occurred before it ended. Values are not applied: value the units with the asset values and " +
			"convert with the rates (ADR 0015). A range of more than 10,000 changes is refused; shorten it.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *SeriesInput) (*SeriesOutput, error) {
		from, err := time.Parse(time.DateOnly, in.From)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("from is not a date, such as 2026-01-01")
		}
		to, err := time.Parse(time.DateOnly, in.To)
		if err != nil {
			return nil, huma.Error422UnprocessableEntity("to is not a date, such as 2026-01-31")
		}
		result, err := gateway.RunWorkflow[lt.NetWorthSeriesResult](ctx, c, reportingwf.NetWorthSeries,
			lt.NetWorthSeriesParams{From: from, To: to})
		if err != nil {
			return nil, err
		}

		out := &SeriesOutput{Body: NetWorthSeries{
			Balances: make([]NetWorthBalance, len(result.Balances)),
			Units:    make([]NetWorthUnits, len(result.Units)),
		}}
		for n, row := range result.Balances {
			out.Body.Balances[n] = NetWorthBalance{
				Date: row.Date.Format(time.DateOnly), CurrencyID: row.CurrencyID,
				Assets: gateway.NewAmount(row.Assets), Liabilities: gateway.NewAmount(row.Liabilities),
			}
		}
		for n, row := range result.Units {
			out.Body.Units[n] = NetWorthUnits{Date: row.Date.Format(time.DateOnly), AssetID: row.AssetID, Units: gateway.NewAmount(row.Units)}
		}
		return out, nil
	})
}
