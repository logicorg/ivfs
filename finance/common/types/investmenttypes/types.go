// Package investmenttypes holds the inputs and results of the intent-level
// investments (ADR 0009): a buy, a sale, and a vest.
package investmenttypes

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// TradeResult is the journal entry and position movement a trade wrote, or
// found by its idempotency key.
type TradeResult struct {
	JournalEntryID     int64 `json:"journalEntryId"`
	PositionMovementID int64 `json:"positionMovementId"`
	// Warnings lists the holdings that are negative after the trade, which was
	// written regardless (ADR 0010).
	Warnings []lt.NegativeHolding `json:"warnings,omitempty"`
}

// TODO: remove this function
// validateOccurredAt checks the instant a trade occurred at, which CUE cannot:
// a zero time is encoded as a valid timestamp, not as an empty value.
func validateOccurredAt(occurredAt time.Time) error {
	if occurredAt.IsZero() {
		return errs.ErrMissingRequiredParameter{Parameters: []string{"OccurredAt"}}
	}
	return nil
}
