package sqllib

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/driver/pgdriver"
)

// foreignKeyViolationCode is the SQLSTATE PostgreSQL raises for a foreign key
// violation, and restrictViolationCode the one PostgreSQL 18 raises instead
// when the violated key is ON DELETE RESTRICT, as every key here is. Earlier
// versions raise foreignKeyViolationCode for both.
const (
	foreignKeyViolationCode = "23503"
	restrictViolationCode   = "23001"
)

func isForeignKeyViolationCode(code string) bool {
	return code == foreignKeyViolationCode || code == restrictViolationCode
}

// violatedKeyValue extracts the value from the detail of a foreign key
// violation, which reads like: Key (asset_id)=(42) is not present in table
// "asset".
var violatedKeyValue = regexp.MustCompile(`=\(([^)]*)\)`)

// foreignKey is a reference from a column of a table to the id of its parent.
type foreignKey struct {
	table  string
	column string
	parent string
	// parentType names the parent in the errors that mention it.
	parentType string
}

// name is the constraint name PostgreSQL would give the key by default.
func (f foreignKey) name() string {
	return f.table[strings.Index(f.table, ".")+1:] + "_" + f.column + "_fkey"
}

// foreignKeys are the references between the tables. They all restrict
// deletion: a row that is still referenced can never be deleted, so history is
// neither destroyed nor left dangling.
var foreignKeys = []foreignKey{
	{"asset.asset_value", "asset_id", "asset.asset", "Asset"},
	{"asset.asset_value", "currency_id", "currency.currency", "Currency"},
	{"currency.conversion", "from_currency_id", "currency.currency", "Currency"},
	{"currency.conversion", "to_currency_id", "currency.currency", "Currency"},
	{"account.financial_account", "institution_id", "institution.institution", "Institution"},
	{"ledger.ledger_account", "parent_id", "ledger.ledger_account", "LedgerAccount"},
	{"ledger.ledger_account", "financial_account_id", "account.financial_account", "FinancialAccount"},
	{"ledger.ledger_account", "currency_id", "currency.currency", "Currency"},
	{"ledger.journal_entry", "counterparty_id", "ledger.counterparty", "Counterparty"},
	{"ledger.posting", "journal_entry_id", "ledger.journal_entry", "JournalEntry"},
	{"ledger.posting", "ledger_account_id", "ledger.ledger_account", "LedgerAccount"},
	{"asset.position_movement", "journal_entry_id", "ledger.journal_entry", "JournalEntry"},
	{"asset.position_movement", "financial_account_id", "account.financial_account", "FinancialAccount"},
	{"asset.position_movement", "asset_id", "asset.asset", "Asset"},
	{"asset.position_movement", "cost_basis_currency_id", "currency.currency", "Currency"},
	{"settings.user_settings", "default_currency_id", "currency.currency", "Currency"},
}

// ensureForeignKeys declares the foreign keys once every table exists. They
// are added afterwards rather than with the tables because the tables refer to
// each other, and because bun only declares them when it creates a table, which
// an existing database would skip. It is safe to call repeatedly.
func ensureForeignKeys(ctx context.Context, db *bun.DB) error {
	for _, fk := range foreignKeys {
		statement := fmt.Sprintf("ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (id) ON DELETE RESTRICT",
			fk.table, fk.name(), fk.column, fk.parent)
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install foreign key %s: %w", fk.name(), err)
		}
	}
	return nil
}

// violatedForeignKey returns the foreign key a PostgreSQL error violates,
// together with the error itself.
func violatedForeignKey(err error) (foreignKey, pgdriver.Error, bool) {
	var pgErr pgdriver.Error
	if !errors.As(err, &pgErr) || !isForeignKeyViolationCode(pgErr.Field('C')) {
		return foreignKey{}, pgErr, false
	}

	constraint := pgErr.Field('n')
	for _, fk := range foreignKeys {
		if fk.name() == constraint {
			return fk, pgErr, true
		}
	}
	return foreignKey{}, pgErr, false
}

// IsForeignKeyViolation reports whether err is a PostgreSQL foreign key
// violation.
func IsForeignKeyViolation(err error) bool {
	var pgErr pgdriver.Error
	return errors.As(err, &pgErr) && isForeignKeyViolationCode(pgErr.Field('C'))
}

// ParentNotFound turns the violation raised by inserting a row that refers to a
// parent that does not exist into an ErrNotFound naming the parent. It returns
// nil for any other error, so a caller can fall back to wrapping err itself. The
// result is meant to be returned as it is: Temporal derives the type of an
// error from its Go type, so wrapping it would hide it from the non-retryable
// error types.
func ParentNotFound(err error) error {
	fk, pgErr, ok := violatedForeignKey(err)
	if !ok {
		return nil
	}

	notFound := internalerrors.ErrNotFound{ResourceType: fk.parentType, Err: fmt.Errorf("referenced by %s.%s", fk.table, fk.column)}
	if match := violatedKeyValue.FindStringSubmatch(pgErr.Field('D')); match != nil {
		notFound.ID = match[1]
	}
	return notFound
}

// StillReferenced turns the violation raised by deleting a row that other rows
// still refer to into an ErrResourceConflict for resourceType. It returns nil
// for any other error. Like ParentNotFound, the result has to be returned as
// it is.
func StillReferenced(err error, resourceType string) error {
	fk, _, ok := violatedForeignKey(err)
	if !ok {
		return nil
	}
	return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: fmt.Sprintf("still referenced by %s.%s", fk.table, fk.column)}
}
