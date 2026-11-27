package deletion

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// LedgerAccountHasPostingsTx reports whether any posting, in any journal entry
// status, references the ledger account.
func LedgerAccountHasPostingsTx(ctx context.Context, db bun.Tx, accountID int64) (bool, error) {
	exists, err := db.NewSelect().Model((*model.Posting)(nil)).Where("ledger_account_id = ?", accountID).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check ledger account postings: %w", err)
	}
	return exists, nil
}
