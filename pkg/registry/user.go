// Package registry is the registry database, ivfs: the one table of users,
// and the name of each user's own database (ADR 0013). It is not part of any
// user's database, so its model stays out of model.Schemas.
package registry

import (
	"encoding/hex"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// User is someone who signs in. It holds who the identity provider says they
// are, and nothing they could sign in with: ivfs stores no credentials.
type User struct {
	bun.BaseModel `bun:"table:iam.user,alias:u"`

	// ID is a UUIDv7, never reused, so a database, workflow ID, or session
	// cookie left from a recreated registry cannot be handed to someone else.
	ID uuid.UUID `bun:"id,pk,type:uuid"`
	// Issuer and Subject are the OIDC identity. Both compare exactly: a
	// subject is case-sensitive, and only unique within its issuer.
	Issuer  string `bun:"issuer,notnull,unique:identity"`
	Subject string `bun:"subject,notnull,unique:identity"`

	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
	// ProvisionedAt is when the user's database was ready, nil until then.
	// Nothing opens a user's database before it is set.
	ProvisionedAt *time.Time `bun:"provisioned_at,nullzero"`
}

// DatabaseName is the name of uid's database. It is derived rather than
// stored, so no row can point at someone else's; hex keeps it an identifier
// that needs no quoting, and 39 characters fit PostgreSQL's 63.
func DatabaseName(uid uuid.UUID) string {
	return "ivfs_u_" + hex.EncodeToString(uid[:])
}
