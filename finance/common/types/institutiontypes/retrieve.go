package institutiontypes

import (
	"fmt"
	"strings"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveInstitutionParams holds the parameters for retrieving an institution.
// Either ID or Name must be provided.
type RetrieveInstitutionParams struct {
	ID   int64  `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Validate checks the validity of the parameters.
func (p *RetrieveInstitutionParams) Validate() error {
	if p.ID <= 0 && strings.TrimSpace(p.Name) == "" {
		return errors.ValidationError{Err: fmt.Errorf("either name or id must be provided")}
	}

	return nil
}

// RetrieveInstitutionResult is the result of a retrieve institution operation.
type RetrieveInstitutionResult struct {
	// Institution is the retrieved institution.
	Institution *model.Institution `json:"institution"`
}
