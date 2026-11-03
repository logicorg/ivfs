package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/logicorg/ivfs/pkg/registry"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

// TODO: simplify concurrency handling (consistent workflow id?)

const (
	duplicateDatabaseCode = "42P04"
	uniqueViolationCode   = "23505"
)

// CreateUserDatabase creates the database of a registered user, and succeeds
// when it is already there.
//
// CREATE DATABASE cannot run in a transaction, so this activity holds none.
// The check against pg_database spares the usual login a failing statement;
// a concurrent login that creates the database between the check and the
// create surfaces as one of the two codes isDuplicateDatabase accepts.
func (a *Activities) CreateUserDatabase(ctx context.Context, uid uuid.UUID) error {
	if _, err := a.retrieve(ctx, uid); err != nil {
		return err
	}

	name := registry.DatabaseName(uid)
	exists, err := a.Registry.NewSelect().TableExpr("pg_database").Where("datname = ?", name).Exists(ctx)
	if err != nil {
		return fmt.Errorf("check for database %s: %w", name, err)
	}
	if exists {
		return nil
	}

	_, err = a.Registry.ExecContext(ctx, "CREATE DATABASE ?", bun.Ident(name))
	if err != nil && !isDuplicateDatabase(err) {
		return fmt.Errorf("create database %s: %w", name, err)
	}
	return nil
}

// isDuplicateDatabase reports whether CREATE DATABASE failed because the
// database exists. PostgreSQL checks the name before it writes the catalog
// row, so a create that loses the race past that check fails on the unique
// index of pg_database instead, after the winner has committed.
func isDuplicateDatabase(err error) bool {
	var pgErr pgdriver.Error
	if !errors.As(err, &pgErr) {
		return false
	}
	code := pgErr.Field('C')
	return code == duplicateDatabaseCode || code == uniqueViolationCode
}
