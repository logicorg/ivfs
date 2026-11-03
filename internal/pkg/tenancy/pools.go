package tenancy

import (
	"context"

	"github.com/uptrace/bun"
)

// Pools hands an activity the database of the user it runs for. Activities
// hold one instead of a *bun.DB, so the database they write to follows the
// context they were given rather than whatever was wired in at registration.
type Pools interface {
	// For returns the connection pool of the user ctx acts as.
	For(ctx context.Context) (*bun.DB, error)
}
