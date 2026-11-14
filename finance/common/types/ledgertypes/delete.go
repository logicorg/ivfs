package ledgertypes

import (
	"cuelang.org/go/cuego"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

// DeleteJournalEntryInput deletes a journal entry of any status, with its
// postings and its position movements (ADR 0011).
// Deleting an entry that does not exist is not an error.
type DeleteJournalEntryInput struct {
	JournalEntryID int64 `json:"journalEntryId" cue:">0"`
}

// DeleteJournalEntryResult carries the holdings that are negative after the
// deletion, which was made regardless.
type DeleteJournalEntryResult struct {
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

// Validate checks the command without the database.
func (i *DeleteJournalEntryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	return nil
}

// DeletePositionMovementInput deletes the movements written by the same command
// as the movement, as ADR 0011 describes. Movements
// of a journal entry are deleted with the entry. Deleting a movement that does
// not exist is not an error.
type DeletePositionMovementInput struct {
	MovementID int64 `json:"movementId" cue:">0"`
}

// DeletePositionMovementResult carries the holdings that are negative after the
// deletion, which was made regardless.
type DeletePositionMovementResult struct {
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}

// Validate checks the command without the database.
func (i *DeletePositionMovementInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	return nil
}
