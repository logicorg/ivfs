package temporaltesting

import (
	"context"
	"testing"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// InsertPostedJournalEntry inserts entry as posted with its postings. The
// database rejects a posted entry that is not balanced, so the entry and its
// postings have to share one transaction.
func InsertPostedJournalEntry(t *testing.T, db *bun.DB, entry *model.JournalEntry, postings ...model.Posting) {
	t.Helper()

	InsertPostedJournalEntryWithMovements(t, db, entry, postings, nil)
}

// InsertPostedJournalEntryWithMovements is InsertPostedJournalEntry for an entry
// that also has position movements. The database only accepts a movement of an
// entry on its date, so the movements take the date of the entry. They are
// completed by WithMovementDefaults and updated in place.
func InsertPostedJournalEntryWithMovements(t *testing.T, db *bun.DB, entry *model.JournalEntry, postings []model.Posting, movements []*model.PositionMovement) {
	t.Helper()

	err := db.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		now := time.Now().UTC()
		entry.Status, entry.PostedAt = model.JournalEntryPosted, &now
		if _, err := tx.NewInsert().Model(entry).Returning("id").Exec(ctx); err != nil {
			return err
		}

		for i := range postings {
			postings[i].JournalEntryID = entry.ID
			postings[i].LineNumber = i + 1
		}
		if _, err := tx.NewInsert().Model(&postings).Exec(ctx); err != nil {
			return err
		}

		for _, movement := range movements {
			movement.JournalEntryID = &entry.ID
			movement.OccurredAt = entry.OccurredAt
			if _, err := tx.NewInsert().Model(WithMovementDefaults(movement)).Returning("id").Exec(ctx); err != nil {
				return err
			}
		}

		return nil
	})
	require.NoError(t, err)
}

// DeleteJournalEntry deletes an entry of any status with its postings, so a
// test can remove what it posted. A posted entry without postings is not
// balanced, so both go in one transaction. An entry with movements is refused.
func DeleteJournalEntry(t *testing.T, db *bun.DB, id int64) {
	t.Helper()

	err := db.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", id).Exec(ctx); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*model.JournalEntry)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	})
	require.NoError(t, err)
}
