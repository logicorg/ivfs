package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// ProvisionUser creates the schema in a user's database, then sets
// provisioned_at, which is what lets tenancy.Pools open it. The schema comes
// first so that a failure in between leaves the user unprovisioned, never
// provisioned with half a schema.
//
// A user already provisioned is left alone: provisioned_at keeps the time it
// was first set, and Pools brings the schema up to date when it opens the
// database.
func (a *Activities) ProvisionUser(ctx context.Context, uid uuid.UUID) error {
	user, err := a.retrieve(ctx, uid)
	if err != nil {
		return err
	}
	if user.ProvisionedAt != nil {
		return nil
	}

	name := registry.DatabaseName(uid)
	db := a.Server.Open(name)
	defer db.Close()
	if err := sqllib.InitDatabase(ctx, db); err != nil {
		return fmt.Errorf("initialise database %s: %w", name, err)
	}

	_, err = a.Registry.NewUpdate().Model((*registry.User)(nil)).
		Set("provisioned_at = current_timestamp").
		Where("id = ?", uid).
		Where("provisioned_at IS NULL").
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("mark user %s provisioned: %w", uid, err)
	}
	return nil
}
