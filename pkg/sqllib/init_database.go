package sqllib

import (
	"context"
	"fmt"
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

const (
	createSchemaQueryFmt = "CREATE SCHEMA IF NOT EXISTS %s"

	errCreateSchemaFmt = "failed to create schema: %w"
	errCreateTableFmt  = "failed to create %s table: %w"
)

// InitDatabase initilise the DB
func InitDatabase(ctx context.Context, db *bun.DB) error {
	db.RegisterModel(
		(*model.FinancialAccount)(nil),
		(*model.LedgerAccount)(nil),
		(*model.Counterparty)(nil),
		(*model.JournalEntry)(nil),
		(*model.Posting)(nil),
		(*model.PositionMovement)(nil),
		(*model.Institution)(nil),
		(*model.Asset)(nil),
		(*model.AssetValue)(nil),
		(*model.Currency)(nil),
		(*model.Conversion)(nil),
		(*model.Tag)(nil),
		(*model.UserSettings)(nil))

	for _, schema := range model.Schemas {
		if _, err := db.ExecContext(ctx, fmt.Sprintf(createSchemaQueryFmt, schema)); err != nil {
			return fmt.Errorf(errCreateSchemaFmt, err)
		}
	}

	_, err := db.NewCreateTable().Model((*model.Currency)(nil)).
		IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "currency", err)
	}

	_, err = db.NewCreateTable().Model((*model.Asset)(nil)).
		IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "asset", err)
	}

	_, err = db.NewCreateTable().Model((*model.AssetValue)(nil)).
		IfNotExists().Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "asset_value", err)
	}

	_, err = db.NewCreateTable().Model((*model.Conversion)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "conversion", err)
	}

	_, err = db.NewCreateTable().Model((*model.Institution)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "institution", err)
	}

	_, err = db.NewCreateTable().Model((*model.FinancialAccount)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "financial_account", err)
	}

	_, err = db.NewCreateTable().Model((*model.Counterparty)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "counterparty", err)
	}

	_, err = db.NewCreateTable().Model((*model.LedgerAccount)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "ledger_account", err)
	}

	_, err = db.NewCreateTable().Model((*model.JournalEntry)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "journal_entry", err)
	}

	_, err = db.NewCreateTable().Model((*model.Posting)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "posting", err)
	}

	_, err = db.NewCreateTable().Model((*model.PositionMovement)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "position_movement", err)
	}
	_, err = db.NewCreateTable().Model((*model.UserSettings)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "user_settings", err)
	}
	_, err = db.NewCreateTable().Model((*model.Tag)(nil)).
		IfNotExists().
		Exec(ctx)
	if err != nil {
		return fmt.Errorf(errCreateTableFmt, "tag", err)
	}
	if err := ensureLedgerConstraints(ctx, db); err != nil {
		return err
	}
	if err := ensurePositionConstraints(ctx, db); err != nil {
		return err
	}
	if err := ensureSettingsConstraints(ctx, db); err != nil {
		return err
	}
	if err := ensureUniqueKeys(ctx, db); err != nil {
		return err
	}
	if err := ensureForeignKeys(ctx, db); err != nil {
		return err
	}

	return nil
}

// InitDatabaseT initialise the DB in tests
func InitDatabaseT(t *testing.T, db *bun.DB) {
	t.Helper()

	if err := InitDatabase(t.Context(), db); err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
}
