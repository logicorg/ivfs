package accounts

import (
	"context"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// CloseFinancialAccount closes a FinancialAccount. It is idempotent: closing
// an already-closed account succeeds without changing ClosedAt. It refuses to
// close an account that still has active (non-closed) linked ledger
// accounts, or while it still holds units of an asset; those must be closed or
// moved out first so a balance is never left stranded under a closed financial
// account.
func (a *Activities) CloseFinancialAccount(ctx context.Context, input lt.CloseFinancialAccountInput) error {
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

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// Holding the account lock keeps a concurrent trade from adding units
		// between the check below and the close.
		locked, err := positions.LockFinancialAccountsTx(ctx, tx, []int64{input.FinancialAccountID})
		if err != nil {
			return err
		}
		account := locked[input.FinancialAccountID]

		if account.ClosedAt != nil {
			return nil // already closed
		}

		activeLedgerAccounts, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).
			Where("la.financial_account_id = ?", account.ID).
			Where("la.closed_at IS NULL").
			Exists(ctx)
		if err != nil {
			return fmt.Errorf("check active linked ledger accounts: %w", err)
		}
		if activeLedgerAccounts {
			return internalerrors.ErrResourceConflict{ResourceType: "FinancialAccount", Reason: "financial account has active linked ledger accounts"}
		}

		var held []positions.Units
		if err := positions.UnitsQuery(tx, positions.Filter{FinancialAccountID: account.ID}).Scan(ctx, &held); err != nil {
			return fmt.Errorf("check held positions: %w", err)
		}
		for _, position := range held {
			if !position.Units.IsZero() {
				return internalerrors.ErrResourceConflict{ResourceType: "FinancialAccount", Reason: fmt.Sprintf("financial account still holds units of asset %d", position.AssetID)}
			}
		}

		account.ClosedAt = &closedAt
		if _, err := tx.NewUpdate().Model(account).Column("closed_at", "updated_at").WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("close financial account: %w", err)
		}
		return nil
	})
}
