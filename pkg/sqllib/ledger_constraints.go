package sqllib

import (
	"context"
	"fmt"
	"strings"

	"github.com/uptrace/bun"
)

const (
	PgErrCodeDuplicateObject = "42710"
)

// ensureLedgerConstraints installs the invariants that span multiple ledger
// rows. It is safe to call repeatedly during local/test bootstrap.
func ensureLedgerConstraints(ctx context.Context, db *bun.DB) error {
	statements := []string{
		`ALTER TABLE ledger.ledger_account ADD CONSTRAINT ledger_account_kind_check CHECK (kind IN ('asset', 'liability', 'equity', 'income', 'expense'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_side_check CHECK (side IN ('debit', 'credit'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_amount_positive_check CHECK (amount > 0)`,
		`ALTER TABLE ledger.journal_entry ADD CONSTRAINT journal_entry_status_check CHECK (status IN ('draft', 'posted'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_line_unique UNIQUE (journal_entry_id, line_number)`,
		// The journal list pages through the entries in date order, and filters the
		// entries by a posting on an account.
		`CREATE INDEX IF NOT EXISTS journal_entry_occurred_at_idx ON ledger.journal_entry (occurred_at, id)`,
		`CREATE INDEX IF NOT EXISTS posting_ledger_account_idx ON ledger.posting (ledger_account_id)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install ledger constraint: %w", err)
		}
	}

	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION ledger.validate_posted_journal_entry() RETURNS trigger AS $$
		DECLARE posting_count integer;
		BEGIN
			IF NEW.status = 'posted' THEN
				SELECT COUNT(*) INTO posting_count FROM ledger.posting WHERE journal_entry_id = NEW.id;
				IF posting_count < 2 THEN
					RAISE EXCEPTION 'posted journal entry % must have at least two postings', NEW.id;
				END IF;
				IF EXISTS (
					SELECT 1 FROM ledger.posting p
					JOIN ledger.ledger_account a ON a.id = p.ledger_account_id
					WHERE p.journal_entry_id = NEW.id
					GROUP BY a.currency_id
					HAVING SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END) <> 0
				) THEN
					RAISE EXCEPTION 'posted journal entry % is not balanced by currency', NEW.id;
				END IF;
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		return fmt.Errorf("install ledger posting validation function: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS validate_posted_journal_entry ON ledger.journal_entry;
		CREATE CONSTRAINT TRIGGER validate_posted_journal_entry
		AFTER INSERT OR UPDATE OF status ON ledger.journal_entry
		DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION ledger.validate_posted_journal_entry()`); err != nil {
		return fmt.Errorf("install ledger posting validation trigger: %w", err)
	}

	// A posting of a posted entry is frozen, with two exceptions: it is deleted
	// with its entry (ADR 0011), and it moves off a ledger account being deleted
	// onto one of the same currency (ADR 0012). The balance check runs only when
	// an entry is posted, so the currency check here is what keeps a moved entry
	// balanced. Comparing the rows without the account catches any other column,
	// including one added later.
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION ledger.prevent_posted_posting_mutation() RETURNS trigger AS $$
		DECLARE entry_id bigint;
		BEGIN
			IF TG_OP = 'DELETE' THEN entry_id = OLD.journal_entry_id; ELSE entry_id = NEW.journal_entry_id; END IF;
			IF TG_OP = 'DELETE' AND current_setting('ivfs.deleting_journal_entry', true) = entry_id::text THEN
				RETURN OLD;
			END IF;
			IF TG_OP = 'UPDATE'
				AND current_setting('ivfs.deleting_ledger_account', true) = OLD.ledger_account_id::text
				AND to_jsonb(NEW) - 'ledger_account_id' = to_jsonb(OLD) - 'ledger_account_id' THEN
				IF (SELECT currency_id FROM ledger.ledger_account WHERE id = NEW.ledger_account_id)
					IS DISTINCT FROM (SELECT currency_id FROM ledger.ledger_account WHERE id = OLD.ledger_account_id) THEN
					RAISE EXCEPTION 'posting % cannot move from ledger account % to %: they have different currencies', OLD.id, OLD.ledger_account_id, NEW.ledger_account_id;
				END IF;
				RETURN NEW;
			END IF;
			IF EXISTS (SELECT 1 FROM ledger.journal_entry WHERE id = entry_id AND status = 'posted') THEN
				RAISE EXCEPTION 'postings for a posted journal entry are immutable';
			END IF;
			IF TG_OP = 'DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		return fmt.Errorf("install ledger immutability function: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS prevent_posted_posting_mutation ON ledger.posting;
		CREATE TRIGGER prevent_posted_posting_mutation
		BEFORE INSERT OR UPDATE OR DELETE ON ledger.posting
		FOR EACH ROW EXECUTE FUNCTION ledger.prevent_posted_posting_mutation()`); err != nil {
		return fmt.Errorf("install ledger immutability trigger: %w", err)
	}

	// Only a draft entry can change or be deleted. Posting it is an update of a
	// draft, so it is allowed, and afterwards the entry is frozen, which makes
	// draft to posted the only transition. An entry of any status can be deleted
	// by a transaction that has declared it is deleting that entry (ADR 0011).
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION ledger.prevent_settled_journal_entry_mutation() RETURNS trigger AS $$
		BEGIN
			IF TG_OP = 'DELETE' AND current_setting('ivfs.deleting_journal_entry', true) = OLD.id::text THEN
				RETURN OLD;
			END IF;
			IF OLD.status <> 'draft' THEN
				RAISE EXCEPTION 'journal entry % is % and cannot be changed or deleted', OLD.id, OLD.status;
			END IF;
			IF TG_OP = 'DELETE' THEN RETURN OLD; ELSE RETURN NEW; END IF;
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		return fmt.Errorf("install journal entry immutability function: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS prevent_settled_journal_entry_mutation ON ledger.journal_entry;
		CREATE TRIGGER prevent_settled_journal_entry_mutation
		BEFORE UPDATE OR DELETE ON ledger.journal_entry
		FOR EACH ROW EXECUTE FUNCTION ledger.prevent_settled_journal_entry_mutation()`); err != nil {
		return fmt.Errorf("install journal entry immutability trigger: %w", err)
	}
	return nil
}

func isDuplicateConstraintError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), PgErrCodeDuplicateObject))
}
