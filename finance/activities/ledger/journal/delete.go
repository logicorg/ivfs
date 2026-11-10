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

// DeleteJournalEntry deletes an entry of any status, its postings, and its
// position movements, in one transaction (ADR 0011). A closed account does not
// prevent it, and an entry that does not exist is not an error, so a retry
// after a delete succeeds.
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
		// A posted entry's postings and movements never change, so the accounts
		// read before the lock are the ones it still touches after it.
		if err := deletion.LockAccountsOfTx(ctx, tx, []int64{input.JournalEntryID}); err != nil {
			return err
		}

		var movements []model.PositionMovement
		if err := tx.NewSelect().Model(&movements).Where("pm.journal_entry_id = ?", input.JournalEntryID).Scan(ctx); err != nil {
			return fmt.Errorf("load position movements: %w", err)
		}
		if err := deletion.DeleteEntryTx(ctx, tx, input.JournalEntryID); err != nil {
			return err
		}

		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(movements))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
