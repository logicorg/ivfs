package networth

import "github.com/logicorg/ivfs/web/api/gateway"

// SeriesInput is the request of GET /api/net-worth/series.
type SeriesInput struct {
	From string `query:"from" format:"date" required:"true" doc:"the first day, such as 2026-01-01"`
	To   string `query:"to" format:"date" required:"true" doc:"the last day, included; from itself for a single day"`
}

// NetWorthBalance is the asset and the liability balances in one currency at the end
// of a day, each in its natural direction.
type NetWorthBalance struct {
	Date        string         `json:"date" format:"date"`
	CurrencyID  int64          `json:"currencyId"`
	Assets      gateway.Amount `json:"assets" doc:"leaves out brokerages' Investments balances, which the units stand for"`
	Liabilities gateway.Amount `json:"liabilities" doc:"what is owed, positive"`
}

// NetWorthUnits is the units held of one asset at the end of a day, across every
// financial account.
type NetWorthUnits struct {
	Date    string         `json:"date" format:"date"`
	AssetID int64          `json:"assetId"`
	Units   gateway.Amount `json:"units" doc:"negative for a short position"`
}

// NetWorthSeries is a sparse series, oldest first: each currency and asset not at
// zero has a row on from, carrying in what was held before it, then one on
// each day its figure changes. Carry a figure forward until its next row; an
// absent one is zero.
type NetWorthSeries struct {
	Balances []NetWorthBalance `json:"balances"`
	Units    []NetWorthUnits   `json:"units"`
}

// SeriesOutput is the response of GET /api/net-worth/series.
type SeriesOutput struct {
	Body NetWorthSeries
}
