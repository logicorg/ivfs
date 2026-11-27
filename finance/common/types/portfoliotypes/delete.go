package portfoliotypes

import (
	"cuelang.org/go/cuego"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

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
	Warnings []lt.NegativeHolding `json:"warnings,omitempty"`
}

// Validate checks the command without the database.
func (i *DeletePositionMovementInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	return nil
}
