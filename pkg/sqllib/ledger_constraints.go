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
		`ALTER TABLE ledger.category ADD CONSTRAINT category_kind_check CHECK (kind IN ('income', 'expense'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_side_check CHECK (side IN ('debit', 'credit'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_amount_positive_check CHECK (amount > 0)`,
		`ALTER TABLE ledger.journal_entry ADD CONSTRAINT journal_entry_status_check CHECK (status IN ('draft', 'posted'))`,
		`ALTER TABLE ledger.posting ADD CONSTRAINT posting_line_unique UNIQUE (journal_entry_id, line_number)`,
		// The journal list pages through the entries in date order, and filters the
		// entries by a posting on an account.
		`CREATE INDEX IF NOT EXISTS journal_entry_occurred_at_idx ON ledger.journal_entry (occurred_at, id)`,
		`CREATE INDEX IF NOT EXISTS posting_ledger_account_idx ON ledger.posting (ledger_account_id)`,
		// Totals, the journal list, and deleting a category all find the
		// postings of a category.
		`CREATE INDEX IF NOT EXISTS posting_category_idx ON ledger.posting (category_id)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install ledger constraint: %w", err)
		}
	}

	// A posted entry has at least two postings and balances in each currency,
	// and a category only on a posting to an account of the category's kind. A
	// category is income or expense, so that also keeps one off every other
	// kind of account. Nothing freezes the entry, so the check runs for every
	// entry a transaction touched, through the entry or through one of its
	// postings, and at commit, when a write made of several statements is
	// complete. An entry that is gone by then, or is not posted, has nothing to
	// check.
	if _, err := db.ExecContext(ctx, `
		CREATE OR REPLACE FUNCTION ledger.validate_posted_journal_entry() RETURNS trigger AS $$
		DECLARE entry_ids bigint[]; entry_id bigint; posting_count integer;
		BEGIN
			IF TG_TABLE_NAME = 'journal_entry' THEN entry_ids = ARRAY[NEW.id];
			ELSIF TG_OP = 'INSERT' THEN entry_ids = ARRAY[NEW.journal_entry_id];
			ELSIF TG_OP = 'DELETE' THEN entry_ids = ARRAY[OLD.journal_entry_id];
			ELSE entry_ids = ARRAY[OLD.journal_entry_id, NEW.journal_entry_id];
			END IF;
			FOREACH entry_id IN ARRAY entry_ids LOOP
				CONTINUE WHEN NOT EXISTS (SELECT 1 FROM ledger.journal_entry WHERE id = entry_id AND status = 'posted');
				SELECT COUNT(*) INTO posting_count FROM ledger.posting WHERE journal_entry_id = entry_id;
				IF posting_count < 2 THEN
					RAISE EXCEPTION 'posted journal entry % must have at least two postings', entry_id;
				END IF;
				IF EXISTS (
					SELECT 1 FROM ledger.posting p
					JOIN ledger.ledger_account a ON a.id = p.ledger_account_id
					WHERE p.journal_entry_id = entry_id
					GROUP BY a.currency_id
					HAVING SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END) <> 0
				) THEN
					RAISE EXCEPTION 'posted journal entry % is not balanced by currency', entry_id;
				END IF;
				IF EXISTS (
					SELECT 1 FROM ledger.posting p
					JOIN ledger.ledger_account a ON a.id = p.ledger_account_id
					JOIN ledger.category c ON c.id = p.category_id
					WHERE p.journal_entry_id = entry_id AND c.kind <> a.kind
				) THEN
					RAISE EXCEPTION 'posted journal entry % has a category on an account of another kind', entry_id;
				END IF;
			END LOOP;
			RETURN NULL;
		END;
		$$ LANGUAGE plpgsql`); err != nil {
		return fmt.Errorf("install ledger posting validation function: %w", err)
	}
	if _, err := db.ExecContext(ctx, `
		DROP TRIGGER IF EXISTS validate_posted_journal_entry ON ledger.journal_entry;
		CREATE CONSTRAINT TRIGGER validate_posted_journal_entry
		AFTER INSERT OR UPDATE OF status ON ledger.journal_entry
		DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION ledger.validate_posted_journal_entry();
		DROP TRIGGER IF EXISTS validate_posted_journal_entry_postings ON ledger.posting;
		CREATE CONSTRAINT TRIGGER validate_posted_journal_entry_postings
		AFTER INSERT OR UPDATE OR DELETE ON ledger.posting
		DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION ledger.validate_posted_journal_entry()`); err != nil {
		return fmt.Errorf("install ledger posting validation trigger: %w", err)
	}
	return nil
}

func isDuplicateConstraintError(err error) bool {
	return err != nil && (strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), PgErrCodeDuplicateObject))
}
