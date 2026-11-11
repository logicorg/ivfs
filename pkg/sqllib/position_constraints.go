package sqllib

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// ensurePositionConstraints installs the invariants of a position movement that
// the columns alone cannot express. It is safe to call repeatedly during
// local/test bootstrap.
func ensurePositionConstraints(ctx context.Context, db *bun.DB) error {
	statements := []string{
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_kind_check CHECK (kind IN ('trade', 'vest', 'split', 'transfer', 'adjustment'))`,
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_units_nonzero_check CHECK (units_delta <> 0)`,
		// Only the kinds with a monetary effect have a journal entry.
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_entry_kind_check CHECK ((kind IN ('trade', 'vest')) = (journal_entry_id IS NOT NULL))`,
		// A movement without an entry is identified by its own key and line.
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_key_line_check CHECK ((idempotency_key IS NULL) = (line_number IS NULL))`,
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_entryless_key_check CHECK (journal_entry_id IS NOT NULL OR idempotency_key IS NOT NULL)`,
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_cost_basis_currency_check CHECK ((cost_basis_amount IS NULL) = (cost_basis_currency_id IS NULL))`,
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_idempotency_line_unique UNIQUE (idempotency_key, line_number)`,
		// The holdings check and the reports read one position in time order.
		`CREATE INDEX IF NOT EXISTS position_movement_position_idx ON asset.position_movement (financial_account_id, asset_id, occurred_at, created_at, id)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install position movement constraint: %w", err)
		}
	}

	return nil
}

// ensureMovementEntryDate makes a movement that has a journal entry share the
// entry's date, and follow it when the date changes. It is declared after the
// plain foreign keys, so that a movement of an entry that does not exist is
// reported through the key on the id alone, which names the entry.
func ensureMovementEntryDate(ctx context.Context, db *bun.DB) error {
	statements := []string{
		`ALTER TABLE ledger.journal_entry ADD CONSTRAINT journal_entry_id_occurred_at_unique UNIQUE (id, occurred_at)`,
		`ALTER TABLE asset.position_movement ADD CONSTRAINT position_movement_entry_date_fkey
			FOREIGN KEY (journal_entry_id, occurred_at) REFERENCES ledger.journal_entry (id, occurred_at)
			ON UPDATE CASCADE ON DELETE RESTRICT`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install position movement entry date: %w", err)
		}
	}
	return nil
}
