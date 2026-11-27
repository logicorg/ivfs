package posting

import (
	"context"
	"fmt"
	"slices"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// SetTagsTx makes tagIDs the tags of the entry, each once, in an existing SQL
// transaction. A tag that does not exist fails with ErrNotFound,
// raised by the foreign key, which also keeps the tag from being deleted
// before the transaction commits.
func SetTagsTx(ctx context.Context, tx bun.Tx, entryID int64, tagIDs []int64) error {
	if _, err := tx.NewDelete().Model((*model.JournalEntryTag)(nil)).Where("journal_entry_id = ?", entryID).Exec(ctx); err != nil {
		return fmt.Errorf("delete journal entry tags: %w", err)
	}

	ids := slices.Compact(slices.Sorted(slices.Values(tagIDs)))
	if len(ids) == 0 {
		return nil
	}
	links := make([]model.JournalEntryTag, 0, len(ids))
	for _, id := range ids {
		links = append(links, model.JournalEntryTag{JournalEntryID: entryID, TagID: id})
	}
	if _, err := tx.NewInsert().Model(&links).Exec(ctx); err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return notFound
		}
		return fmt.Errorf("insert journal entry tags: %w", err)
	}
	return nil
}
