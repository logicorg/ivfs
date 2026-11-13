package posting

import (
	"context"
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// RewriteEntryTx replaces what a correction changes in a stored entry: its
// date, description, and metadata from the input, and its postings and
// movements, which are deleted and written again. The row stays, so the entry
// keeps its id, its keys, its counterparty, and its times, and the rest of the
// input is ignored. The caller holds the lock on the entry's row, and entry is
// the row as read under it; it is updated in place.
//
// The movements take the entry's creation time, which is their place among
// the movements of the same date, and its external id, as a movement recorded
// with an entry does.
//
// touchedLedgerAccountIDs are the ledger accounts the entry had postings on. A
// correction may leave a posting on one of them that has since been closed,
// which a new entry may not.
func RewriteEntryTx(ctx context.Context, tx bun.Tx, entry *model.JournalEntry, input lt.PostJournalEntryInput, movements []model.PositionMovement, touchedLedgerAccountIDs []int64) error {
	if err := checkPostingsTx(ctx, tx, input, touchedLedgerAccountIDs); err != nil {
		return err
	}

	if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("journal_entry_id = ?", entry.ID).Exec(ctx); err != nil {
		return fmt.Errorf("delete position movements: %w", err)
	}
	if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", entry.ID).Exec(ctx); err != nil {
		return fmt.Errorf("delete journal postings: %w", err)
	}

	entry.OccurredAt, entry.Description, entry.Metadata = input.OccurredAt, input.Description, input.Metadata
	entry.UpdatedAt = time.Now().UTC()
	if _, err := tx.NewUpdate().Model(entry).Column("occurred_at", "description", "metadata", "updated_at").WherePK().Exec(ctx); err != nil {
		return fmt.Errorf("update journal entry: %w", err)
	}

	for i := range movements {
		movements[i].CreatedAt = entry.CreatedAt
		movements[i].ExternalID = entry.ExternalID
	}
	return insertLinesTx(ctx, tx, entry, input.Postings, movements)
}
