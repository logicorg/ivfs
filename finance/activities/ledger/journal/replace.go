package journal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// ReplaceJournalEntry corrects a posted entry in one transaction: it updates
// the entry's date, description, and metadata, and replaces its postings and
// its position movements with those of the input. The entry keeps its id, its
// keys, its counterparty, and its times.
//
// The entry may stay on a closed account it already touches, since closing is
// final and a mistake still needs correcting, but it may not move onto one.
//
// The holdings that are negative afterwards, among the positions of the old
// and the new movements, are returned as warnings. A retry writes the same
// state again, so it needs no idempotency key.
func (a *Activities) ReplaceJournalEntry(ctx context.Context, input lt.ReplaceJournalEntryInput) (*lt.ReplaceJournalEntryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	entryInput := input.Entry()
	newFinancialIDs := make([]int64, 0, len(input.Movements))
	for _, m := range input.Movements {
		newFinancialIDs = append(newFinancialIDs, m.FinancialAccountID)
	}

	result := &lt.ReplaceJournalEntryResult{JournalEntryID: input.JournalEntryID}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// The accounts are locked before the entry, as every writer does, so
		// they are read once without a lock and again under it.
		financialIDs, ledgerIDs, err := deletion.AccountsOfTx(ctx, tx, []int64{input.JournalEntryID})
		if err != nil {
			return err
		}
		lockedFinancialIDs, lockedLedgerIDs := slices.Concat(financialIDs, newFinancialIDs), slices.Concat(ledgerIDs, entryInput.AccountIDs())
		if err := deletion.LockAccountsTx(ctx, tx, lockedFinancialIDs, lockedLedgerIDs); err != nil {
			return err
		}

		var entry model.JournalEntry
		if err := tx.NewSelect().Model(&entry).Where("je.id = ?", input.JournalEntryID).For("UPDATE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return internalerrors.ErrNotFound{ResourceType: "JournalEntry", ID: input.JournalEntryID}
			}
			return fmt.Errorf("lock journal entry: %w", err)
		}
		if entry.Status != model.JournalEntryPosted {
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("journal entry %d is not posted, and a replace would post it", entry.ID)}
		}

		var oldMovements []model.PositionMovement
		if err := tx.NewSelect().Model(&oldMovements).Where("pm.journal_entry_id = ?", entry.ID).Scan(ctx); err != nil {
			return fmt.Errorf("load position movements: %w", err)
		}
		_, touchedLedgerIDs, err := deletion.AccountsOfTx(ctx, tx, []int64{entry.ID})
		if err != nil {
			return err
		}
		touchedFinancialIDs := deletion.AccountIDsOf(oldMovements)
		// Another replace of the entry may have committed between the first read
		// and the lock, and moved the entry onto accounts this transaction does
		// not hold. Failing lets the activity run again from what is there now.
		heldLedger := func(id int64) bool { return slices.Contains(lockedLedgerIDs, id) }
		heldFinancial := func(id int64) bool { return slices.Contains(lockedFinancialIDs, id) }
		if slices.ContainsFunc(touchedLedgerIDs, func(id int64) bool { return !heldLedger(id) }) ||
			slices.ContainsFunc(touchedFinancialIDs, func(id int64) bool { return !heldFinancial(id) }) {
			return fmt.Errorf("journal entry %d changed while its accounts were being locked", entry.ID)
		}

		// Already locked above, so this only reads them.
		newAccounts, err := positions.LockFinancialAccountsTx(ctx, tx, newFinancialIDs)
		if err != nil {
			return err
		}
		movements := make([]model.PositionMovement, 0, len(input.Movements))
		for _, m := range input.Movements {
			if newAccounts[m.FinancialAccountID].ClosedAt != nil && !slices.Contains(touchedFinancialIDs, m.FinancialAccountID) {
				return internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: m.FinancialAccountID}
			}
			movements = append(movements, model.PositionMovement{
				Kind:                m.Kind,
				FinancialAccountID:  m.FinancialAccountID,
				AssetID:             m.AssetID,
				UnitsDelta:          m.UnitsDelta,
				CostBasisAmount:     m.CostBasisAmount,
				CostBasisCurrencyID: m.CostBasisCurrencyID,
			})
		}

		if err := posting.RewriteEntryTx(ctx, tx, &entry, entryInput, movements, touchedLedgerIDs); err != nil {
			return err
		}

		result.PositionMovementIDs = positions.IDs(movements)
		result.Warnings, err = positions.NegativeHoldingsTx(ctx, tx, positions.PairsOf(slices.Concat(oldMovements, movements)))
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
