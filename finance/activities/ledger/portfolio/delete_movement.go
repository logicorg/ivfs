package portfolio

import (
	"context"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/uptrace/bun"
)

// DeletePositionMovement deletes the movements an entry-less command wrote, in
// one transaction (ADR 0011). Any movement of the group identifies it. A movement
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
		group, err := deletion.MovementGroupTx(ctx, tx, input.MovementID)
		if err != nil || len(group) == 0 {
			return err
		}

		if _, err := positions.LockFinancialAccountsTx(ctx, tx, deletion.AccountIDsOf(group)); err != nil {
			return err
		}
		if err := deletion.DeleteGroupTx(ctx, tx, *group[0].IdempotencyKey); err != nil {
			return err
		}

		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(group))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
