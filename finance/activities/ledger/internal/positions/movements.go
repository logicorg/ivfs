package positions

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// FindByIdempotencyKeyTx returns the movements written under an idempotency key,
// in line order, or none when the key is unused. Only movements without a
// journal entry have a key of their own.
func FindByIdempotencyKeyTx(ctx context.Context, tx bun.Tx, key string) ([]model.PositionMovement, error) {
	var movements []model.PositionMovement
	if err := tx.NewSelect().Model(&movements).Where("pm.idempotency_key = ?", key).OrderExpr("pm.line_number").Scan(ctx); err != nil {
		return nil, fmt.Errorf("find position movements by idempotency key: %w", err)
	}
	return movements, nil
}

// IDs returns the ids of the movements, in their order.
func IDs(movements []model.PositionMovement) []int64 {
	ids := make([]int64, len(movements))
	for n := range movements {
		ids[n] = movements[n].ID
	}
	return ids
}

// EffectiveMovementsQuery selects the movements that count towards holdings,
// which are those without a journal entry and those of a posted one, under the
// alias pm. It uses the predicate of the units queries, so a history and a
// position cannot disagree about which movements they count.
func EffectiveMovementsQuery(db bun.IDB) *bun.SelectQuery {
	return db.NewSelect().
		Model((*model.PositionMovement)(nil)).
		Join("LEFT JOIN ledger.journal_entry AS je ON je.id = pm.journal_entry_id").
		Where(effectiveMovement)
}

// PairsOf returns the position of each movement, in their order.
func PairsOf(movements []model.PositionMovement) []Pair {
	pairs := make([]Pair, len(movements))
	for n := range movements {
		pairs[n] = Pair{FinancialAccountID: movements[n].FinancialAccountID, AssetID: movements[n].AssetID}
	}
	return pairs
}
