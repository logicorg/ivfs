package usertypes

import "github.com/google/uuid"

// EnsureUserParams is the identity a login verified.
type EnsureUserParams struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

// EnsureUserResult is the user the identity belongs to, whose database is
// ready to open.
type EnsureUserResult struct {
	ID uuid.UUID `json:"id"`
}
