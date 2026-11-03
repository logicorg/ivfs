package sqllib

import (
	"context"
	"fmt"

	"github.com/uptrace/bun"
)

// ensureSettingsConstraints keeps the settings to one row and the gap policy
// to the ones that are built (ADR 0015). Widening the gap policy check means
// dropping it first: an existing database keeps the old one otherwise. It is
// safe to call repeatedly.
func ensureSettingsConstraints(ctx context.Context, db *bun.DB) error {
	statements := []string{
		`ALTER TABLE settings.user_settings ADD CONSTRAINT user_settings_single_row_check CHECK (id = 1)`,
		`ALTER TABLE settings.user_settings ADD CONSTRAINT user_settings_gap_policy_check CHECK (gap_policy IN ('exclude'))`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil && !isDuplicateConstraintError(err) {
			return fmt.Errorf("install settings constraint: %w", err)
		}
	}
	return nil
}
