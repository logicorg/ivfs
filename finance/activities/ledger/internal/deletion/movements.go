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

// MovementGroupsTx returns the group of the movement, then the group that
// reverses it, then the one that reverses that, and so on, or nothing when the
// movement does not exist. It rejects a movement that belongs to a journal
// entry.
func MovementGroupsTx(ctx context.Context, tx bun.Tx, movementID int64) ([][]model.PositionMovement, error) {
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

	var groups [][]model.PositionMovement
	// A movement without an entry always has a key, which the database checks.
	group, err := positions.FindByIdempotencyKeyTx(ctx, tx, *movement.IdempotencyKey)
	for err == nil && len(group) > 0 {
		groups = append(groups, group)

		var reversal model.PositionMovement
		err = tx.NewSelect().Model(&reversal).Where("pm.reversal_of_movement_id IN (?)", bun.List(positions.IDs(group))).Limit(1).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return groups, nil
		}
		if err != nil {
			return nil, fmt.Errorf("load reversing position movement: %w", err)
		}
		group, err = positions.FindByIdempotencyKeyTx(ctx, tx, *reversal.IdempotencyKey)
	}
	return groups, err
}

// AccountIDsOf returns the financial accounts the groups move units in.
func AccountIDsOf(groups [][]model.PositionMovement) []int64 {
	var ids []int64
	for _, group := range groups {
		for _, movement := range group {
			ids = append(ids, movement.FinancialAccountID)
		}
	}
	return ids
}

// DeleteGroupTx deletes the entry-less movements written under key, declaring
// the group to the trigger that guards them. The setting ends with the
// transaction.
func DeleteGroupTx(ctx context.Context, tx bun.Tx, key string) error {
	if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_position_movement_group', ?, true)`, key); err != nil {
		return fmt.Errorf("declare position movement group deletion: %w", err)
	}
	if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("idempotency_key = ?", key).Where("journal_entry_id IS NULL").Exec(ctx); err != nil {
		return fmt.Errorf("delete position movements: %w", err)
	}
	return nil
}
