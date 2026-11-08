package sqllib

import (
	"context"
	"errors"
	"fmt"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

// uniqueKeys are the unique indexes that make a name identify one row within
// its scope, compared case-insensitively (ADR 0005). They back the lookup each
// creation activity does against a concurrent create of the same name. They
// are named apart from the column constraints they replace, so that an older
// database still holding one cannot make CREATE INDEX IF NOT EXISTS skip them.
var uniqueKeys = []string{
	`CREATE UNIQUE INDEX IF NOT EXISTS institution_name_lower_key ON institution.institution (lower(name))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS financial_account_name_lower_key ON account.financial_account (institution_id, lower(name))`,
	// external_id identifies a real account, such as an IBAN; an account
	// without one is NULL and never collides.
	`CREATE UNIQUE INDEX IF NOT EXISTS financial_account_external_id_key ON account.financial_account (institution_id, external_id)`,
	`CREATE UNIQUE INDEX IF NOT EXISTS ledger_account_financial_account_name_lower_key
		ON ledger.ledger_account (financial_account_id, currency_id, lower(name))
		WHERE financial_account_id IS NOT NULL`,
	// A top-level account has no parent, and NULLS NOT DISTINCT makes two
	// top-level accounts with the same kind, currency, and name collide, which
	// they would not by default. It needs PostgreSQL 15 or later.
	`CREATE UNIQUE INDEX IF NOT EXISTS ledger_account_name_lower_key
		ON ledger.ledger_account (parent_id, kind, currency_id, lower(name)) NULLS NOT DISTINCT
		WHERE financial_account_id IS NULL`,
	`CREATE UNIQUE INDEX IF NOT EXISTS counterparty_name_lower_key ON ledger.counterparty (lower(name))`,
	`CREATE UNIQUE INDEX IF NOT EXISTS tag_name_lower_key ON tag.tag (lower(name))`,
}

func ensureUniqueKeys(ctx context.Context, db *bun.DB) error {
	for _, statement := range uniqueKeys {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("install unique key: %w", err)
		}
	}
	return nil
}

// uniqueViolationCode is the SQLSTATE PostgreSQL raises for a unique
// violation.
const uniqueViolationCode = "23505"

// AlreadyExists turns the violation raised by writing a row whose unique key
// another row already holds into an ErrResourceConflict for resourceType, with
// PostgreSQL's detail - "Key (code)=(USD) already exists." - as the reason. It
// returns nil for any other error, so a caller can fall back to wrapping err
// itself. Like StillReferenced, the result has to be returned as it is.
func AlreadyExists(err error, resourceType string) error {
	var pgErr pgdriver.Error
	if !errors.As(err, &pgErr) || pgErr.Field('C') != uniqueViolationCode {
		return nil
	}
	return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: pgErr.Field('D')}
}

func IsAlreadyExists(err error) bool {
	var pgErr pgdriver.Error
	if !errors.As(err, &pgErr) || pgErr.Field('C') != uniqueViolationCode {
		return false
	}
	return true
}
