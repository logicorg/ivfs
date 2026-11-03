// Package tenancy carries the signed-in user from the gateway to the
// activities that open their database (ADR 0013). The user never travels as
// a workflow or activity argument: it lives in a Go or workflow context, under
// a key only this package can set, and crosses Temporal in a header.
package tenancy

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// userKey is unexported so that a user can enter a context only through
// WithUser, which lint allows only where the session is verified, or through
// the propagator, which trusts the network boundary.
type userKey struct{}

// WithUser returns a copy of ctx that acts as uid. Only the gateway's
// authentication middleware and tests may call it: anywhere else it would let
// code act as a user it was not handed.
func WithUser(ctx context.Context, uid uuid.UUID) context.Context {
	return context.WithValue(ctx, userKey{}, uid)
}

// UserFrom returns the user ctx acts as. The nil UUID counts as no user, so
// a zero value that slipped through never opens a database.
func UserFrom(ctx context.Context) (uuid.UUID, bool) {
	return user(ctx.Value(userKey{}))
}

// WorkflowUser returns the user a workflow runs as, which the propagator
// extracted from its header.
func WorkflowUser(ctx workflow.Context) (uuid.UUID, bool) {
	return user(ctx.Value(userKey{}))
}

func user(v any) (uuid.UUID, bool) {
	uid, ok := v.(uuid.UUID)
	if !ok || uid == uuid.Nil {
		return uuid.Nil, false
	}
	return uid, true
}
