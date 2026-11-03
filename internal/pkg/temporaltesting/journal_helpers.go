package temporaltesting

import (
	"context"
	"testing"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// InsertPostedJournalEntry inserts entry with its postings and marks it as
// posted. The database rejects a posted entry that is not balanced, so the
// entry, its postings, and the status change have to share one transaction.
func InsertPostedJournalEntry(t *testing.T, db *bun.DB, entry *model.JournalEntry, postings ...model.Posting) {
	t.Helper()

	InsertPostedJournalEntryWithMovements(t, db, entry, postings, nil)
}

// InsertPostedJournalEntryWithMovements is InsertPostedJournalEntry for an entry
// that also has position movements. The database only accepts a movement of an
// entry while it is still a draft, and on its date, so the movements are
// inserted between its postings and the status change, take the date of the
// entry, and are completed by WithMovementDefaults. They are updated in place.
func InsertPostedJournalEntryWithMovements(t *testing.T, db *bun.DB, entry *model.JournalEntry, postings []model.Posting, movements []*model.PositionMovement) {
	t.Helper()

	err := db.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		entry.Status = model.JournalEntryDraft
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

		now := time.Now().UTC()
		entry.Status, entry.PostedAt = model.JournalEntryPosted, &now
		_, err := tx.NewUpdate().Model(entry).Column("status", "posted_at", "updated_at").WherePK().Exec(ctx)
		return err
	})
	require.NoError(t, err)
}
