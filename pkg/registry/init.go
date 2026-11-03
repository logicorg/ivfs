package registry

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// Init creates the registry's schema and table, and leaves them alone when
// they are there.
func Init(ctx context.Context, db *bun.DB) error {
	if _, err := db.ExecContext(ctx, "CREATE SCHEMA IF NOT EXISTS iam"); err != nil {
		return fmt.Errorf("create schema iam: %w", err)
	}
	if _, err := db.NewCreateTable().Model((*User)(nil)).IfNotExists().Exec(ctx); err != nil {
		return fmt.Errorf("create table iam.user: %w", err)
	}
	return nil
}
