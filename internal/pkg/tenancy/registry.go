package tenancy

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// RegistryPools hands out each user's own database, as the registry knows
// them (ADR 0013). It fails closed: a context without a user, or with one
// whose database is not provisioned, gets no database at all.
type RegistryPools struct {
	registry *bun.DB
	server   registry.Server

	mu    sync.Mutex
	pools map[uuid.UUID]*pool
}

// pool is one user's database, opened on first use. Its lock serialises the
// first opens, since two InitDatabase running on one database can race each
// other's CREATE ... IF NOT EXISTS into a unique violation.
type pool struct {
	mu sync.Mutex
	db *bun.DB
}

// NewRegistryPools returns Pools that look users up in the registry db and
// open their databases on server.
func NewRegistryPools(db *bun.DB, server registry.Server) *RegistryPools {
	return &RegistryPools{registry: db, server: server, pools: make(map[uuid.UUID]*pool)}
}

// For returns the database of the user ctx acts as, opening it the first
// time. An open database stays cached: provisioned_at is never unset, so a
// database that was ready stays ready.
func (p *RegistryPools) For(ctx context.Context) (*bun.DB, error) {
	uid, ok := UserFrom(ctx)
	if !ok {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"user"}}
	}

	p.mu.Lock()
	userPool, ok := p.pools[uid]
	if !ok {
		userPool = &pool{}
		p.pools[uid] = userPool
	}
	p.mu.Unlock()

	userPool.mu.Lock()
	defer userPool.mu.Unlock()
	if userPool.db == nil {
		db, err := p.open(ctx, uid)
		if err != nil {
			return nil, err
		}
		userPool.db = db
	}
	return userPool.db, nil
}

// open checks that uid is provisioned, then brings its schema up to date,
// which is how a deploy that adds a table reaches existing users.
func (p *RegistryPools) open(ctx context.Context, uid uuid.UUID) (*bun.DB, error) {
	user := &registry.User{ID: uid}
	err := p.registry.NewSelect().Model(user).WherePK().Scan(ctx)
	switch {
	case sqllib.IsNotFound(err):
		return nil, internalerrors.ErrNotFound{ResourceType: "User", ID: uid}
	case err != nil:
		return nil, fmt.Errorf("retrieve user %s: %w", uid, err)
	case user.ProvisionedAt == nil:
		return nil, internalerrors.ErrResourceConflict{
			ResourceType: "User",
			Reason:       fmt.Sprintf("the database of user %s is not provisioned yet; signing in again finishes it", uid),
		}
	}

	name := registry.DatabaseName(uid)
	db := p.server.Open(name)
	if err := sqllib.InitDatabase(ctx, db); err != nil {
		return nil, errors.Join(fmt.Errorf("initialise database %s: %w", name, err), db.Close())
	}
	return db, nil
}

// Close closes every database For opened.
func (p *RegistryPools) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	var errs []error
	for _, userPool := range p.pools {
		userPool.mu.Lock()
		if userPool.db != nil {
			errs = append(errs, userPool.db.Close())
		}
		userPool.mu.Unlock()
	}
	return errors.Join(errs...)
}
