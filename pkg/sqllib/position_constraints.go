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

	// A movement is append-only: it is never changed, and a correction deletes it
	// and records it again. It is deleted only by a transaction that has declared it is
	// deleting the movement's journal entry, or, for a movement without one, its
	// idempotency key group (ADR 0011). A transaction deleting a financial
	// account can move its movements onto another account, and delete its
	// entry-less splits and adjustments alone (ADR 0012). One with a journal entry
	// can only be inserted while that entry is still a draft, on the entry's own
	// date, which is how the entry, its postings, and its movements are written as
	// one.
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION asset.guard_position_movement() RETURNS trigger AS $$
		DECLARE entry_status text; entry_time timestamptz;
		BEGIN
			IF TG_OP = 'UPDATE'
				AND current_setting('ivfs.deleting_financial_account', true) = OLD.financial_account_id::text
				AND to_jsonb(NEW) - 'financial_account_id' = to_jsonb(OLD) - 'financial_account_id' THEN
				RETURN NEW;
			END IF;
			IF TG_OP = 'DELETE' AND OLD.journal_entry_id IS NULL AND OLD.kind IN ('split', 'adjustment')
				AND current_setting('ivfs.deleting_financial_account', true) = OLD.financial_account_id::text THEN
				RETURN OLD;
			END IF;
			IF TG_OP = 'DELETE' THEN
				IF OLD.journal_entry_id IS NOT NULL AND current_setting('ivfs.deleting_journal_entry', true) = OLD.journal_entry_id::text THEN
					RETURN OLD;
				END IF;
				IF OLD.journal_entry_id IS NULL AND current_setting('ivfs.deleting_position_movement_group', true) = OLD.idempotency_key THEN
					RETURN OLD;
				END IF;
			END IF;
			IF TG_OP <> 'INSERT' THEN
				RAISE EXCEPTION 'position movement % is append-only', OLD.id;
			END IF;
			IF NEW.journal_entry_id IS NOT NULL THEN
				SELECT status, occurred_at INTO entry_status, entry_time FROM ledger.journal_entry WHERE id = NEW.journal_entry_id;
				IF entry_status IS DISTINCT FROM 'draft' THEN
					RAISE EXCEPTION 'a position movement can only be attached to a draft journal entry (entry %)', NEW.journal_entry_id;
				END IF;
				IF entry_time <> NEW.occurred_at THEN
					RAISE EXCEPTION 'a position movement must share the date of its journal entry (entry %)', NEW.journal_entry_id;
				END IF;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		return fmt.Errorf("install position movement guard function: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS guard_position_movement ON asset.position_movement;
		CREATE TRIGGER guard_position_movement
		BEFORE INSERT OR UPDATE OR DELETE ON asset.position_movement
		FOR EACH ROW EXECUTE FUNCTION asset.guard_position_movement()`); err != nil {
		return fmt.Errorf("install position movement guard trigger: %w", err)
	}
	return nil
}
