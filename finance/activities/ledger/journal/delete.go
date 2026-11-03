package journal

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// DeleteJournalEntry deletes an entry of any status, its postings, its position
// movements, and the chain of reversals that follow it, in one transaction
// (ADR 0011). Deleting a reversal alone is allowed and puts the effect of the
// entry it reversed back. A closed account does not prevent it, and an entry
// that does not exist is not an error, so a retry after a delete succeeds.
//
// The units an entry gave are gone from the history afterwards, so the holdings
// that are negative as a result are returned as warnings.
func (a *Activities) DeleteJournalEntry(ctx context.Context, input lt.DeleteJournalEntryInput) (*lt.DeleteJournalEntryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.DeleteJournalEntryResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		chain, err := deletion.ReversalChainTx(ctx, tx, input.JournalEntryID)
		if err != nil || len(chain) == 0 {
			return err
		}

		// A reversal written meanwhile touches the same accounts, so reading the
		// chain again under the locks is what makes it complete.
		if err := deletion.LockAccountsOfTx(ctx, tx, chain); err != nil {
			return err
		}
		if chain, err = deletion.ReversalChainTx(ctx, tx, input.JournalEntryID); err != nil || len(chain) == 0 {
			return err
		}

		var movements []model.PositionMovement
		if err := tx.NewSelect().Model(&movements).Where("pm.journal_entry_id IN (?)", bun.List(chain)).Scan(ctx); err != nil {
			return fmt.Errorf("load position movements: %w", err)
		}

		// A reversal refers to what it reverses, so the last one goes first.
		for n := len(chain) - 1; n >= 0; n-- {
			if err := deletion.DeleteEntryTx(ctx, tx, chain[n]); err != nil {
				return err
			}
		}

		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(movements))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
