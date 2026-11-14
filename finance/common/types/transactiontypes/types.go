// Package transactiontypes holds the inputs and results of the intent-level
// transactions (ADR 0008): an expense, an income, and a transfer.
package transactiontypes

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// RecordResult is the journal entry a transaction posted, or found by its
// idempotency key.
type RecordResult struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// UpdateResult is the journal entry an update corrected. Recording a
// transaction never warns, but an update can remove the units of an entry that
// was a trade, so it carries the holdings left negative.
type UpdateResult struct {
	JournalEntryID int64                `json:"journalEntryId"`
	Warnings       []lt.NegativeHolding `json:"warnings,omitempty"`
}

// validateOccurredAt checks the instant a transaction occurred at, which CUE
// cannot: a zero time is encoded as a valid timestamp, not as an empty value.
func validateOccurredAt(occurredAt time.Time) error {
	if occurredAt.IsZero() {
		return errs.ErrMissingRequiredParameter{Parameters: []string{"OccurredAt"}}
	}
	return nil
}
