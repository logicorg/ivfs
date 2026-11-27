package accounts

import (
	"context"
	"fmt"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	"github.com/uptrace/bun"
	"time"
)

// CloseLedgerAccount closes a LedgerAccount, at ClosedAt or now if it is zero.
// It is idempotent: closing a closed account succeeds without changing
// ClosedAt. It refuses to close an account with open children, or an asset or
// liability that still has a balance.
func (a *Activities) CloseLedgerAccount(ctx context.Context, input act.CloseLedgerAccountInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	closedAt := input.ClosedAt
	if closedAt.IsZero() {
		closedAt = time.Now().UTC()
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account, err := loadLedgerAccount(ctx, tx, input.LedgerAccountID)
		if err != nil {
			return err
		}

		if account.ClosedAt != nil {
			return nil // already closed
		}

		if err := refuseStrandedBalance(ctx, tx, account); err != nil {
			return err
		}

		account.ClosedAt = &closedAt
		if _, err := tx.NewUpdate().Model(account).Column("closed_at", "updated_at").WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("failed to close ledger account: %w", err)
		}
		return nil
	})
	return err
}
