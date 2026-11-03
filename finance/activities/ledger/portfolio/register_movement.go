package portfolio

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// RegisterPositionMovement records a change of units that has no cash effect
// (a split, a transfer between accounts, or an adjustment) as one atomic write
// of all its legs. Retrying with the same idempotency key returns the rows of
// the first call without comparing the rest of the request. A holding that is
// negative afterwards does not refuse the movement and is returned as a
// warning.
func (a *Activities) RegisterPositionMovement(ctx context.Context, input lt.RegisterPositionMovementInput) (*lt.RegisterPositionMovementResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.RegisterPositionMovementResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		accounts, err := positions.LockFinancialAccountsTx(ctx, tx, input.FinancialAccountIDs())
		if err != nil {
			return err
		}

		existing, err := positions.FindByIdempotencyKeyTx(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return err
		}
		if len(existing) > 0 {
			if existing[0].ReversalOfMovementID != nil {
				return internalerrors.ErrResourceConflict{ResourceType: "PositionMovement", Reason: "idempotency key belongs to a reversal"}
			}
			result.PositionMovementIDs = positions.IDs(existing)
			result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(existing))
			return err
		}

		for _, leg := range input.Legs {
			if accounts[leg.FinancialAccountID].ClosedAt != nil {
				return internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: leg.FinancialAccountID}
			}
		}

		movements := input.ToMovements()
		if _, err := tx.NewInsert().Model(&movements).Returning("id").Exec(ctx); err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return notFound
			}
			return fmt.Errorf("insert position movements: %w", err)
		}

		result.PositionMovementIDs = positions.IDs(movements)
		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(movements))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
