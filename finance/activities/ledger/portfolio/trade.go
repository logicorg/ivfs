package portfolio

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

type Activities struct{ Pools tenancy.Pools }

// RegisterInvestmentTrade atomically posts the cash entry and records the
// signed units movement. The entry must contain the cash-side postings. A
// holding that is negative afterwards does not refuse the trade and is returned
// as a warning.
func (a *Activities) RegisterInvestmentTrade(ctx context.Context, input lt.RegisterInvestmentTradeInput) (*lt.RegisterInvestmentTradeResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.RegisterInvestmentTradeResult{}
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
			if err := tx.NewSelect().Model(&movement).Where("journal_entry_id = ?", existing.ID).Scan(ctx); err != nil {
				if err == sql.ErrNoRows {
					return internalerrors.ErrResourceConflict{ResourceType: "JournalEntry", Reason: "idempotency key belongs to an entry without an investment movement"}
				}
				return fmt.Errorf("load existing position movement: %w", err)
			}
			result.JournalEntryID = existing.ID
			result.PositionMovementID = movement.ID
			result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf([]model.PositionMovement{movement}))
			return err
		}

		if accounts[input.FinancialAccountID].ClosedAt != nil {
			return internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: input.FinancialAccountID}
		}
		var asset model.Asset
		if err := tx.NewSelect().Model(&asset).Where("id = ?", input.AssetID).Scan(ctx); err != nil {
			if err == sql.ErrNoRows {
				return internalerrors.ErrNotFound{ResourceType: "Asset", ID: input.AssetID}
			}
			return fmt.Errorf("load asset: %w", err)
		}

		movements := []model.PositionMovement{{
			Kind:                model.PositionMovementTrade,
			FinancialAccountID:  input.FinancialAccountID,
			AssetID:             input.AssetID,
			UnitsDelta:          input.UnitsDelta,
			CostBasisAmount:     input.CostBasisAmount,
			CostBasisCurrencyID: input.CostBasisCurrencyID,
			ExternalID:          input.Entry.ExternalID,
		}}
		entryID, err := posting.PostEntryWithMovementsTx(ctx, tx, input.Entry, movements)
		if err != nil {
			return err
		}
		result.JournalEntryID = entryID
		result.PositionMovementID = movements[0].ID
		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(movements))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
