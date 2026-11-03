package portfolio

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// RegisterVest atomically posts the entry that books the compensation of a vest
// and records the units received. Retrying with the same idempotency key
// returns the entry and the movement of the first call without comparing the
// rest of the request.
func (a *Activities) RegisterVest(ctx context.Context, input lt.RegisterVestInput) (*lt.RegisterVestResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.RegisterVestResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// The lock comes first so that a concurrent identical request waits and
		// then finds the entry, instead of racing to insert it.
		accounts, err := positions.LockFinancialAccountsTx(ctx, tx, []int64{input.FinancialAccountID})
		if err != nil {
			return err
		}

		existing, err := posting.FindEntryByIdempotencyKeyTx(ctx, tx, input.Entry.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			var movement model.PositionMovement
			if err := tx.NewSelect().Model(&movement).Column("id", "kind").Where("journal_entry_id = ?", existing.ID).Scan(ctx); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return internalerrors.ErrResourceConflict{ResourceType: "JournalEntry", Reason: "idempotency key belongs to an entry without a vest movement"}
				}
				return fmt.Errorf("load existing position movement: %w", err)
			}
			if movement.Kind != model.PositionMovementVest {
				return internalerrors.ErrResourceConflict{ResourceType: "JournalEntry", Reason: "idempotency key belongs to an entry that is not a vest"}
			}
			result.JournalEntryID = existing.ID
			result.PositionMovementID = movement.ID
			return nil
		}

		if accounts[input.FinancialAccountID].ClosedAt != nil {
			return internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: input.FinancialAccountID}
		}

		movements := []model.PositionMovement{{
			Kind:                model.PositionMovementVest,
			FinancialAccountID:  input.FinancialAccountID,
			AssetID:             input.AssetID,
			UnitsDelta:          input.Units,
			CostBasisAmount:     &input.CostBasisAmount,
			CostBasisCurrencyID: &input.CostBasisCurrencyID,
			ExternalID:          input.Entry.ExternalID,
		}}
		entryID, err := posting.PostEntryWithMovementsTx(ctx, tx, input.Entry, movements)
		if err != nil {
			return err
		}

		// A vest only adds units, so it cannot make a position negative and needs
		// no holdings check.
		result.JournalEntryID = entryID
		result.PositionMovementID = movements[0].ID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
