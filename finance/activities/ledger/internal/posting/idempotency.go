package posting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// FindEntryByIdempotencyKeyTx returns the journal entry stored under key, or
// nil when there is none. An empty key never matches.
func FindEntryByIdempotencyKeyTx(ctx context.Context, tx bun.Tx, key string) (*model.JournalEntry, error) {
	if key == "" {
		return nil, nil
	}

	entry := new(model.JournalEntry)
	err := tx.NewSelect().Model(entry).Where("idempotency_key = ?", key).Scan(ctx)
	switch {
	case err == nil:
		return entry, nil
	case errors.Is(err, sql.ErrNoRows):
		return nil, nil
	default:
		return nil, fmt.Errorf("look up journal entry idempotency key: %w", err)
	}
}
