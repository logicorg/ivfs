// Package usertypes holds the inputs and results of the provisioning
// activities.
package usertypes

import (
	"cuelang.org/go/cuego"
	"github.com/google/uuid"
	"github.com/logicorg/ivfs/internal/pkg/errors"
)

// InsertUserParams is an OIDC identity, as a verified ID token states it.
// Neither part is trimmed or folded: identities compare exactly.
type InsertUserParams struct {
	Issuer  string `json:"issuer" cue:"!~\"^[[:space:]]*$\""`
	Subject string `json:"subject" cue:"!~\"^[[:space:]]*$\""`
}

// Validate rejects an identity with a blank issuer or subject.
func (p *InsertUserParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}

// InsertUserResult is the user an identity belongs to. Provisioned lets a
// login of a user whose database is ready skip the steps that make it so.
type InsertUserResult struct {
	ID          uuid.UUID `json:"id"`
	Provisioned bool      `json:"provisioned"`
}
