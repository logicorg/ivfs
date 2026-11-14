package deletion

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// MovementGroupTx returns the movements written by the same command as the
// movement, or nothing when the movement does not exist. It rejects a movement
// that belongs to a journal entry.
func MovementGroupTx(ctx context.Context, tx bun.Tx, movementID int64) ([]model.PositionMovement, error) {
	var movement model.PositionMovement
	if err := tx.NewSelect().Model(&movement).Where("pm.id = ?", movementID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load position movement: %w", err)
	}
	if movement.JournalEntryID != nil {
		return nil, internalerrors.ErrInvalidParameter{Description: "the movement belongs to a journal entry, delete the journal entry instead"}
	}
	// A movement without an entry always has a key, which the database checks.
	return positions.FindByIdempotencyKeyTx(ctx, tx, *movement.IdempotencyKey)
}

// AccountIDsOf returns the financial accounts the movements move units in.
func AccountIDsOf(movements []model.PositionMovement) []int64 {
	ids := make([]int64, 0, len(movements))
	for _, movement := range movements {
		ids = append(ids, movement.FinancialAccountID)
	}
	return ids
}

// DeleteGroupTx deletes the entry-less movements written under key.
func DeleteGroupTx(ctx context.Context, tx bun.Tx, key string) error {
	if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("idempotency_key = ?", key).Where("journal_entry_id IS NULL").Exec(ctx); err != nil {
		return fmt.Errorf("delete position movements: %w", err)
	}
	return nil
}
