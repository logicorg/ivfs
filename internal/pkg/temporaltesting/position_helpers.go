package temporaltesting

import (
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

var movementFixtureSequence atomic.Int64

// WithMovementDefaults fills in what a movement needs to be valid but a test
// rarely cares about: a kind, which is a trade when there is a journal entry and
// an adjustment otherwise, and, for a movement without an entry, an idempotency
// key of its own. Anything the test sets is kept.
func WithMovementDefaults(movement *model.PositionMovement) *model.PositionMovement {
	if movement.Kind == "" {
		movement.Kind = model.PositionMovementAdjustment
		if movement.JournalEntryID != nil {
			movement.Kind = model.PositionMovementTrade
		}
	}
	if movement.JournalEntryID == nil && movement.IdempotencyKey == nil {
		key := fmt.Sprintf("fixture-movement-%d", movementFixtureSequence.Add(1))
		movement.IdempotencyKey, movement.LineNumber = &key, 1
	}
	return movement
}

// InsertPositionMovement inserts a movement, completed by WithMovementDefaults.
func InsertPositionMovement(t *testing.T, db *bun.DB, movement *model.PositionMovement) {
	t.Helper()

	Insert(t, db, WithMovementDefaults(movement))
}
