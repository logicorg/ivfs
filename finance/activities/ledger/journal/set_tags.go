package journal

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// SetJournalEntryTags replaces the tags of an entry, and none clears them. A
// tag records nothing the ledger balances, so the entry is locked against a
// concurrent replace of its tags but its accounts are not. The same input
// leaves the same tags, so a retry needs no key.
func (a *Activities) SetJournalEntryTags(ctx context.Context, input jt.SetJournalEntryTagsInput) (*jt.SetJournalEntryTagsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var entry model.JournalEntry
		if err := tx.NewSelect().Model(&entry).Column("je.id").Where("je.id = ?", input.JournalEntryID).For("UPDATE").Scan(ctx); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return internalerrors.ErrNotFound{ResourceType: "JournalEntry", ID: input.JournalEntryID}
			}
			return fmt.Errorf("lock journal entry: %w", err)
		}
		return posting.SetTagsTx(ctx, tx, entry.ID, input.TagIDs)
	})
	if err != nil {
		return nil, err
	}
	return &jt.SetJournalEntryTagsResult{TagIDs: slices.Compact(slices.Sorted(slices.Values(input.TagIDs)))}, nil
}
