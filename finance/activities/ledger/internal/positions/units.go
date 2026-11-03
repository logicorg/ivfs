// Package positions holds the queries shared by the activities that report or
// check units held, so they all agree on which movements count.
package positions

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// effectiveMovement selects the movements that count towards holdings, given
// the movement aliased pm and its journal entry, if any, aliased je. It is
// shared so the running-balance check and the reports cannot disagree.
const effectiveMovement = "(pm.journal_entry_id IS NULL OR je.status = 'posted')"

// Filter narrows the movements a units query counts. A zero field is ignored.
type Filter struct {
	// FinancialAccountID is the ID of the FinancialAccount to filter on.
	FinancialAccountID int64
	// AssetID is the Asset ID.
	AssetID int64
	// At includes the movements that occurred at or before it.
	At time.Time
}

// Units is the units held of one asset in one financial account.
type Units struct {
	FinancialAccountID int64           `bun:"financial_account_id"`
	AssetID            int64           `bun:"asset_id"`
	Units              decimal.Decimal `bun:"units"`
}

// UnitsQuery sums the effective movements per financial account and asset. A
// movement is effective when it has no journal entry, or when its entry is
// posted, so movements of draft entries never count.
func UnitsQuery(db bun.IDB, filter Filter) *bun.SelectQuery {
	query := db.NewSelect().
		TableExpr("asset.position_movement AS pm").
		ColumnExpr("pm.financial_account_id, pm.asset_id, COALESCE(SUM(pm.units_delta), 0) AS units").
		Join("LEFT JOIN ledger.journal_entry AS je ON je.id = pm.journal_entry_id").
		Where(effectiveMovement).
		GroupExpr("pm.financial_account_id, pm.asset_id").
		OrderExpr("pm.financial_account_id, pm.asset_id")

	if filter.FinancialAccountID > 0 {
		query = query.Where("pm.financial_account_id = ?", filter.FinancialAccountID)
	}
	if filter.AssetID > 0 {
		query = query.Where("pm.asset_id = ?", filter.AssetID)
	}
	if !filter.At.IsZero() {
		query = query.Where("pm.occurred_at <= ?", filter.At)
	}
	return query
}
