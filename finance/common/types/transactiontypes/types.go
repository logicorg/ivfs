// Package transactiontypes holds the inputs and results of the intent-level
// transactions (ADR 0008): an expense, an income, and a transfer.
package transactiontypes

import (
	"time"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// RecordResult is the journal entry a transaction posted, or found by its
// idempotency key.
type RecordResult struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// validateOccurredAt checks the instant a transaction occurred at, which CUE
// cannot: a zero time is encoded as a valid timestamp, not as an empty value.
func validateOccurredAt(occurredAt time.Time) error {
	if occurredAt.IsZero() {
		return errs.ErrMissingRequiredParameter{Parameters: []string{"OccurredAt"}}
	}
	return nil
}
