package journal

import (
	"context"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/posting"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/uptrace/bun"
)

// PostJournalEntry validates and atomically posts a balanced journal entry.
// Retrying with the same idempotency key returns the original entry.
func (a *Activities) PostJournalEntry(ctx context.Context, input jt.PostJournalEntryInput) (*jt.PostJournalEntryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if input.OccurredAt.IsZero() {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"OccurredAt"}}
	}
	if len(input.Postings) < 2 {
		return nil, internalerrors.ErrInvalidParameter{Description: "journal entry requires at least two postings"}
	}

	result := &jt.PostJournalEntryResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		existing, err := posting.FindEntryByIdempotencyKeyTx(ctx, tx, input.IdempotencyKey)
		if err != nil {
			return err
		}
		if existing != nil {
			result.JournalEntryID = existing.ID
			return nil
		}

		entryID, err := posting.PostJournalEntryTx(ctx, tx, input)
		if err != nil {
			return err
		}
		result.JournalEntryID = entryID
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
