package accounts

import (
	"context"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// DeleteLedgerAccount is an activity that deletes a ledger account.
// If the account has postings, or has child accounts, it will return an error indicating that the account cannot be deleted.
func (a *Activities) DeleteLedgerAccount(ctx context.Context, input lt.DeleteLedgerAccountInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := loadLedgerAccount(ctx, tx, input.LedgerAccountID); err != nil {
			return err
		}

		if hasPostings, err := ledgerAccountHasPostings(ctx, tx, input.LedgerAccountID); err != nil {
			return err
		} else if hasPostings {
			return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account has postings; close it instead"}
		}

		children, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.parent_id = ?", input.LedgerAccountID).Exists(ctx)
		if err != nil {
			return fmt.Errorf("check child ledger accounts: %w", err)
		}
		if children {
			return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account has children"}
		}

		if _, err := tx.NewDelete().Model((*model.LedgerAccount)(nil)).Where("id = ?", input.LedgerAccountID).Exec(ctx); err != nil {
			if conflict := sqllib.StillReferenced(err, "LedgerAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("delete ledger account: %w", err)
		}
		return nil
	})
	return err
}
