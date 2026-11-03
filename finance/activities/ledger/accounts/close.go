package accounts

import (
	"context"
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// CloseLedgerAccount closes a LedgerAccount, at ClosedAt or now if it is zero.
// It is idempotent: closing a closed account succeeds without changing
// ClosedAt. It refuses to close an account with open children, or an asset or
// liability that still has a balance.
func (a *Activities) CloseLedgerAccount(ctx context.Context, input lt.CloseLedgerAccountInput) error {
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

		activeChildren, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.parent_id = ?", account.ID).Where("la.closed_at IS NULL").Exists(ctx)
		if err != nil {
			return fmt.Errorf("failed to check active child ledger accounts: %w", err)
		}

		if activeChildren {
			return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account has active children"}
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
