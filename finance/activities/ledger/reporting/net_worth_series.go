package reporting

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// netWorthBalance is a NetWorthSeries balances row.
type netWorthBalance struct {
	Date        time.Time       `bun:"date"`
	CurrencyID  int64           `bun:"currency_id"`
	Assets      decimal.Decimal `bun:"assets"`
	Liabilities decimal.Decimal `bun:"liabilities"`
}

// netWorthUnits is a NetWorthSeries units row.
type netWorthUnits struct {
	Date    time.Time       `bun:"date"`
	AssetID int64           `bun:"asset_id"`
	Units   decimal.Decimal `bun:"units"`
}

// NetWorthSeries returns, for each day from From to To, the asset and the
// liability balances per currency and the units held per asset, as a sparse
// series (ADR 0015). Both lists come from one snapshot, so a trade cannot show
// its cash without its units.
func (a *Activities) NetWorthSeries(ctx context.Context, params lt.NetWorthSeriesParams) (*lt.NetWorthSeriesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}
	// The days are UTC days; the dates go in as text so the session's time
	// zone cannot shift them.
	from := assetvalue.Day(params.From).Format(time.DateOnly)
	end := assetvalue.Day(params.To).AddDate(0, 0, 1)

	var balances []netWorthBalance
	var units []netWorthUnits
	opts := &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}
	err = db.RunInTx(ctx, opts, func(ctx context.Context, tx bun.Tx) error {
		if err := netWorthBalancesQuery(tx, from, end).Scan(ctx, &balances); err != nil {
			return fmt.Errorf("net worth balances: %w", err)
		}
		if err := netWorthUnitsQuery(tx, from, end).Scan(ctx, &units); err != nil {
			return fmt.Errorf("net worth units: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if rows := len(balances) + len(units); rows > lt.NetWorthSeriesMaxRows {
		return nil, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the range holds %d changes, more than the %d a series carries; shorten it", rows, lt.NetWorthSeriesMaxRows)}
	}

	result := &lt.NetWorthSeriesResult{
		Balances: make([]lt.NetWorthBalance, len(balances)),
		Units:    make([]lt.NetWorthUnits, len(units)),
	}
	for n, row := range balances {
		result.Balances[n] = lt.NetWorthBalance{Date: row.Date.UTC(), CurrencyID: row.CurrencyID, Assets: row.Assets, Liabilities: row.Liabilities}
	}
	for n, row := range units {
		result.Units[n] = lt.NetWorthUnits{Date: row.Date.UTC(), AssetID: row.AssetID, Units: row.Units}
	}
	return result, nil
}

// netWorthBalancesQuery sums the posted postings of asset and liability
// accounts per currency and UTC day, folding everything before from into from.
func netWorthBalancesQuery(tx bun.Tx, from string, end time.Time) *bun.SelectQuery {
	deltas := tx.NewSelect().
		TableExpr("ledger.posting AS p").
		Join("JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id").
		Join("JOIN ledger.ledger_account AS la ON la.id = p.ledger_account_id").
		ColumnExpr("GREATEST((je.occurred_at AT TIME ZONE 'UTC')::date, CAST(? AS date)) AS day", from).
		ColumnExpr("la.currency_id").
		ColumnExpr("SUM(CASE WHEN la.kind = ? THEN (CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END) ELSE 0 END) AS assets", model.LedgerAccountAsset).
		ColumnExpr("SUM(CASE WHEN la.kind = ? THEN (CASE WHEN p.side = 'credit' THEN p.amount ELSE -p.amount END) ELSE 0 END) AS liabilities", model.LedgerAccountLiability).
		Where("je.status = ?", model.JournalEntryPosted).
		Where("la.kind IN (?, ?)", model.LedgerAccountAsset, model.LedgerAccountLiability).
		Apply(model.WhereNotBrokerageInvestments).
		Where("je.occurred_at < ?", end).
		GroupExpr("day, la.currency_id")

	running := tx.NewSelect().
		TableExpr("deltas").
		ColumnExpr("day, currency_id").
		ColumnExpr("assets AS assets_delta, liabilities AS liabilities_delta").
		ColumnExpr("SUM(assets) OVER (PARTITION BY currency_id ORDER BY day) AS assets").
		ColumnExpr("SUM(liabilities) OVER (PARTITION BY currency_id ORDER BY day) AS liabilities")

	return tx.NewSelect().
		With("deltas", deltas).
		With("running", running).
		TableExpr("running").
		ColumnExpr("(day::timestamp AT TIME ZONE 'UTC') AS date, currency_id, assets, liabilities").
		Where("(day = CAST(? AS date) AND (assets <> 0 OR liabilities <> 0)) OR (day > CAST(? AS date) AND (assets_delta <> 0 OR liabilities_delta <> 0))", from, from).
		OrderExpr("day, currency_id")
}

// netWorthUnitsQuery sums the effective movements per asset and UTC day,
// across financial accounts, folding everything before from into from.
func netWorthUnitsQuery(tx bun.Tx, from string, end time.Time) *bun.SelectQuery {
	deltas := positions.EffectiveMovementsQuery(tx).
		ColumnExpr("GREATEST((pm.occurred_at AT TIME ZONE 'UTC')::date, CAST(? AS date)) AS day", from).
		ColumnExpr("pm.asset_id").
		ColumnExpr("SUM(pm.units_delta) AS units").
		Where("pm.occurred_at < ?", end).
		GroupExpr("day, pm.asset_id")

	running := tx.NewSelect().
		TableExpr("deltas").
		ColumnExpr("day, asset_id, units AS units_delta").
		ColumnExpr("SUM(units) OVER (PARTITION BY asset_id ORDER BY day) AS units")

	return tx.NewSelect().
		With("deltas", deltas).
		With("running", running).
		TableExpr("running").
		ColumnExpr("(day::timestamp AT TIME ZONE 'UTC') AS date, asset_id, units").
		Where("(day = CAST(? AS date) AND units <> 0) OR (day > CAST(? AS date) AND units_delta <> 0)", from, from).
		OrderExpr("day, asset_id")
}
