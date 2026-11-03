// Package user provisions users: their row in the registry, their database,
// and its schema (ADR 0013). Unlike every finance activity these take the uid
// as an argument, since they run before the user has a database to route to.
package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// Activities provisions users.
type Activities struct {
	// Registry is the registry database.
	Registry *bun.DB
	// Server opens the users' databases, on the server the registry is on.
	Server registry.Server
}

// retrieve returns the registered user uid, so that no database is created
// or initialised for a uid the registry never handed out.
func (a *Activities) retrieve(ctx context.Context, uid uuid.UUID) (*registry.User, error) {
	if uid == uuid.Nil {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"uid"}}
	}
	user := &registry.User{ID: uid}
	err := a.Registry.NewSelect().Model(user).WherePK().Scan(ctx)
	switch {
	case sqllib.IsNotFound(err):
		return nil, internalerrors.ErrNotFound{ResourceType: "User", ID: uid}
	case err != nil:
		return nil, fmt.Errorf("retrieve user %s: %w", uid, err)
	default:
		return user, nil
	}
}
