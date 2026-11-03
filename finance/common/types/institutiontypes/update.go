package institutiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
)

// UpdateInstitutionParams is the input to an update institution operation.
type UpdateInstitutionParams struct {
	FindBy RetrieveInstitutionParams `json:"find_by"`
	Name   string                    `json:"name"`
	Kind   string                    `json:"kind"`
}

// UpdateInstitutionResult is the result of an update institution operation.
type UpdateInstitutionResult struct {
	ID int64 `json:"id"`
}

type CloseParams struct {
	FindBy   RetrieveInstitutionParams `json:"find_by"`
	ClosedAt time.Time                 `json:"closed_at" cue:"!=\"\""`
}

func (p *CloseParams) Validate() error {
	err := p.FindBy.Validate()
	if err != nil {
		return err
	}

	err = cuego.Validate(p)
	if err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}
