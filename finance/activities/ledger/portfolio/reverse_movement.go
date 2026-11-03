package portfolio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// ReversePositionMovement reverses an entry-less movement together with the
// rest of the movements its command wrote, as one atomic write of the opposite
// movements. Retrying with the same idempotency key returns the reversing
// movements of the first call without comparing the rest of the request.
//
// The reversal is allowed on a closed financial account, since it corrects
// history. When the units it restores have been used since, it is written and
// the holdings that are negative afterwards are returned as warnings.
func (a *Activities) ReversePositionMovement(ctx context.Context, input lt.ReversePositionMovementInput) (*lt.ReversePositionMovementResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}

	result := &lt.ReversePositionMovementResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// The movements of a command never change, so the group can be read before
		// its accounts are locked, and the lock is what makes the checks after it
		// hold until the reversal is written.
		group, err := loadReversibleGroup(ctx, tx, input.MovementID)
		if err != nil {
			return err
		}

		pairs := make([]positions.Pair, len(group))
		accountIDs := make([]int64, len(group))
		for n := range group {
			pairs[n] = positions.Pair{FinancialAccountID: group[n].FinancialAccountID, AssetID: group[n].AssetID}
			accountIDs[n] = group[n].FinancialAccountID
		}
		if _, err := positions.LockFinancialAccountsTx(ctx, tx, accountIDs); err != nil {
			return err
		}

		existing, err := positions.FindByIdempotencyKeyTx(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			if existing[0].ReversalOfMovementID == nil {
				return internalerrors.ErrResourceConflict{ResourceType: "PositionMovement", Reason: "idempotency key belongs to movements that are not a reversal"}
			}
			result.PositionMovementIDs = positions.IDs(existing)
			result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(existing))
			return err
		}

		if err := ensureNotReversed(ctx, tx, group); err != nil {
			return err
		}

		reversals := make([]model.PositionMovement, len(group))
		for n := range group {
			reversals[n] = reverseMovement(group[n], input)
		}
		if _, err := tx.NewInsert().Model(&reversals).Returning("id").Exec(ctx); err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return notFound
			}
			return fmt.Errorf("insert reversing position movements: %w", err)
		}

		result.PositionMovementIDs = positions.IDs(reversals)
		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, pairs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// loadReversibleGroup returns the movements written by the same command as the
// movement, in line order. It rejects a movement that does not exist and one
// that belongs to a journal entry.
func loadReversibleGroup(ctx context.Context, tx bun.Tx, movementID int64) ([]model.PositionMovement, error) {
	var movement model.PositionMovement
	if err := tx.NewSelect().Model(&movement).Where("pm.id = ?", movementID).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerrors.ErrNotFound{ResourceType: "PositionMovement", ID: movementID}
		}
		return nil, fmt.Errorf("load position movement: %w", err)
	}
	if movement.JournalEntryID != nil {
		return nil, internalerrors.ErrInvalidParameter{Description: "the movement belongs to a journal entry, reverse the journal entry instead"}
	}

	// A movement without an entry always has a key, which the database checks.
	return positions.FindByIdempotencyKeyTx(ctx, tx, *movement.IdempotencyKey)
}

func ensureNotReversed(ctx context.Context, tx bun.Tx, group []model.PositionMovement) error {
	reversed, err := tx.NewSelect().Model((*model.PositionMovement)(nil)).
		Where("pm.reversal_of_movement_id IN (?)", bun.List(positions.IDs(group))).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("check position movements are not reversed: %w", err)
	}
	if reversed {
		return internalerrors.ErrResourceConflict{ResourceType: "PositionMovement", Reason: "the movement has already been reversed"}
	}
	return nil
}

// reverseMovement returns the movement that cancels the original. Like the
// reversal of a journal entry, it keeps the kind, copies neither the external
// ID nor the date of the original, and is dated when it is made.
func reverseMovement(original model.PositionMovement, input lt.ReversePositionMovementInput) model.PositionMovement {
	reversal := model.PositionMovement{
		Kind:                 original.Kind,
		FinancialAccountID:   original.FinancialAccountID,
		AssetID:              original.AssetID,
		UnitsDelta:           original.UnitsDelta.Neg(),
		CostBasisCurrencyID:  original.CostBasisCurrencyID,
		OccurredAt:           input.OccurredAt,
		LineNumber:           original.LineNumber,
		ReversalOfMovementID: &original.ID,
	}
	key := input.IdempotencyKey
	reversal.IdempotencyKey = &key
	if original.CostBasisAmount != nil {
		negated := original.CostBasisAmount.Neg()
		reversal.CostBasisAmount = &negated
	}
	return reversal
}
