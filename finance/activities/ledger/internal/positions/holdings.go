package positions

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// Pair identifies one position: an asset held in a financial account.
type Pair struct {
	FinancialAccountID int64
	AssetID            int64
}

// LockFinancialAccountsTx locks the financial accounts, in ascending id order
// so concurrent writers cannot deadlock, and returns them by id. Writers to
// the same account are serialised on this lock, which a plain child insert
// does not conflict with, and a concurrent update of the account, such as
// closing it, waits for the transaction. It returns errors.ErrNotFound when an
// account does not exist.
func LockFinancialAccountsTx(ctx context.Context, tx bun.Tx, ids []int64) (map[int64]*model.FinancialAccount, error) {
	unique := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		unique[id] = struct{}{}
	}

	locked := make(map[int64]*model.FinancialAccount, len(unique))
	if len(unique) == 0 {
		return locked, nil
	}

	wanted := make([]int64, 0, len(unique))
	for id := range unique {
		wanted = append(wanted, id)
	}

	var accounts []model.FinancialAccount
	if err := tx.NewSelect().Model(&accounts).Where("fa.id IN (?)", bun.List(wanted)).OrderExpr("fa.id").For("NO KEY UPDATE").Scan(ctx); err != nil {
		return nil, fmt.Errorf("lock financial accounts: %w", err)
	}
	for i := range accounts {
		locked[accounts[i].ID] = &accounts[i]
	}

	for _, id := range wanted {
		if _, ok := locked[id]; !ok {
			return nil, internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: id}
		}
	}
	return locked, nil
}

// CheckHoldingsTx verifies that none of the positions is negative at any point
// in time, counting the effective movements ordered by when they occurred and
// then by id. Movements have to be entered in chronological order for this to
// hold, and a movement inserted after an equal timestamp is judged after it. It
// returns errors.ErrResourceConflict naming the first negative point.
//
// The caller has to hold the lock of the financial accounts and to have
// written the movements in the same transaction.
func CheckHoldingsTx(ctx context.Context, tx bun.Tx, pairs []Pair) error {
	checked := make(map[Pair]struct{}, len(pairs))
	for _, pair := range pairs {
		if _, ok := checked[pair]; ok {
			continue
		}
		checked[pair] = struct{}{}

		var firstNegative struct {
			OccurredAt time.Time       `bun:"occurred_at"`
			Running    decimal.Decimal `bun:"running"`
		}

		// The running sum of the effective movements, taken in the order they
		// happened, is the number of units held right after each one. The position
		// is valid only if none of those balances is negative, so a backdated sale
		// is judged against what was held on its own date rather than against the
		// current total, and a movement that only pushes a later point below zero
		// is refused as well. Ties on occurred_at fall back to id, which is the
		// order of insertion. The frame is spelled out so that each row adds only
		// its own delta. Only the earliest negative balance is fetched, since it is
		// all the error needs to name. Movements of draft entries are left out by
		// the same predicate the reports use.
		err := tx.NewRaw(`WITH ordered AS (
				SELECT pm.id, pm.occurred_at,
					SUM(pm.units_delta) OVER (ORDER BY pm.occurred_at, pm.id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running
				FROM asset.position_movement pm
				LEFT JOIN ledger.journal_entry je ON je.id = pm.journal_entry_id
				WHERE pm.financial_account_id = ? AND pm.asset_id = ? AND `+effectiveMovement+`)
			SELECT occurred_at, running FROM ordered WHERE running < 0 ORDER BY occurred_at, id LIMIT 1`,
			pair.FinancialAccountID, pair.AssetID).Scan(ctx, &firstNegative)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return fmt.Errorf("check holdings: %w", err)
		}

		return internalerrors.ErrResourceConflict{
			ResourceType: "Position",
			Reason: fmt.Sprintf("holdings of asset %d in financial account %d would be negative (%s units) at %s",
				pair.AssetID, pair.FinancialAccountID, firstNegative.Running, firstNegative.OccurredAt.Format(time.RFC3339)),
		}
	}
	return nil
}

// NegativeHoldingsTx returns, for each of the positions that is negative at
// some point in time, the first such point: the date of the movement that takes
// the running total below zero and the units held right after it. It counts the
// effective movements in the same order as CheckHoldingsTx, and a negative
// holding is a finding, not an error. The result is ordered by financial
// account and then by asset, and a position that is never negative is left out.
//
// The caller has to hold the lock of the financial accounts and to have
// written the movements in the same transaction.
func NegativeHoldingsTx(ctx context.Context, tx bun.Tx, pairs []Pair) ([]lt.NegativeHolding, error) {
	unique := make(map[Pair]struct{}, len(pairs))
	for _, pair := range pairs {
		unique[pair] = struct{}{}
	}
	ordered := make([]Pair, 0, len(unique))
	for pair := range unique {
		ordered = append(ordered, pair)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].FinancialAccountID != ordered[j].FinancialAccountID {
			return ordered[i].FinancialAccountID < ordered[j].FinancialAccountID
		}
		return ordered[i].AssetID < ordered[j].AssetID
	})

	var warnings []lt.NegativeHolding
	for _, pair := range ordered {
		var firstNegative struct {
			OccurredAt           time.Time       `bun:"occurred_at"`
			Running              decimal.Decimal `bun:"running"`
			FinancialAccountName string          `bun:"financial_account_name"`
			AssetName            string          `bun:"asset_name"`
		}

		// The running sum is the one CheckHoldingsTx takes, so the two agree on
		// where a position first goes negative. Only the earliest such balance is
		// fetched, since a warning names one point per position, together with the
		// names of the account and the asset so the warning can be read as it is.
		err := tx.NewRaw(`WITH ordered AS (
				SELECT pm.id, pm.occurred_at,
					SUM(pm.units_delta) OVER (ORDER BY pm.occurred_at, pm.id ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running
				FROM asset.position_movement pm
				LEFT JOIN ledger.journal_entry je ON je.id = pm.journal_entry_id
				WHERE pm.financial_account_id = ? AND pm.asset_id = ? AND `+effectiveMovement+`)
			SELECT o.occurred_at, o.running, fa.name AS financial_account_name, a.name AS asset_name
			FROM ordered o
			JOIN account.financial_account fa ON fa.id = ?
			JOIN asset.asset a ON a.id = ?
			WHERE o.running < 0 ORDER BY o.occurred_at, o.id LIMIT 1`,
			pair.FinancialAccountID, pair.AssetID, pair.FinancialAccountID, pair.AssetID).Scan(ctx, &firstNegative)
		switch {
		case errors.Is(err, sql.ErrNoRows):
			continue
		case err != nil:
			return nil, fmt.Errorf("find negative holdings: %w", err)
		}

		warnings = append(warnings, lt.NegativeHolding{
			FinancialAccountID:   pair.FinancialAccountID,
			FinancialAccountName: firstNegative.FinancialAccountName,
			AssetID:              pair.AssetID,
			AssetName:            firstNegative.AssetName,
			At:                   firstNegative.OccurredAt,
			Units:                firstNegative.Running,
		})
	}
	return warnings, nil
}
