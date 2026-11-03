package portfolio

import (
	"context"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// DeletePositionMovement deletes the movements an entry-less command wrote, and
// the groups that reverse them, the reversals of those, and so on, in one
// transaction (ADR 0011). Any movement of the group identifies it. A movement
// that belongs to a journal entry is refused, since it is deleted with its
// entry, and one that does not exist is not an error, so a retry after a delete
// succeeds. A closed financial account does not prevent it.
//
// The units are gone from the history afterwards, so the holdings that are
// negative as a result are returned as warnings.
func (a *Activities) DeletePositionMovement(ctx context.Context, input lt.DeletePositionMovementInput) (*lt.DeletePositionMovementResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.DeletePositionMovementResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		groups, err := deletion.MovementGroupsTx(ctx, tx, input.MovementID)
		if err != nil || len(groups) == 0 {
			return err
		}

		// A reversal written meanwhile touches the same accounts, so reading the
		// groups again under the lock is what makes them complete.
		if _, err := positions.LockFinancialAccountsTx(ctx, tx, deletion.AccountIDsOf(groups)); err != nil {
			return err
		}
		if groups, err = deletion.MovementGroupsTx(ctx, tx, input.MovementID); err != nil || len(groups) == 0 {
			return err
		}

		// A reversal refers to what it reverses, so the last group goes first.
		var deleted []model.PositionMovement
		for n := len(groups) - 1; n >= 0; n-- {
			if err := deletion.DeleteGroupTx(ctx, tx, *groups[n][0].IdempotencyKey); err != nil {
				return err
			}
			deleted = append(deleted, groups[n]...)
		}

		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(deleted))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
