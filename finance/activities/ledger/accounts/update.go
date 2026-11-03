package accounts

import (
	"context"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

func (a *Activities) RenameLedgerAccount(ctx context.Context, input lt.RenameLedgerAccountInput) (*lt.LedgerAccountMutationResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)

	result := &lt.LedgerAccountMutationResult{LedgerAccountID: input.LedgerAccountID}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account, err := loadLedgerAccount(ctx, tx, input.LedgerAccountID)
		if err != nil {
			return err
		}
		account.Name = input.Name
		if _, err := tx.NewUpdate().Model(account).Column("name", "updated_at").WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, "LedgerAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("rename ledger account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (a *Activities) MoveLedgerAccount(ctx context.Context, input lt.MoveLedgerAccountInput) (*lt.LedgerAccountMutationResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.LedgerAccountMutationResult{LedgerAccountID: input.LedgerAccountID}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account, err := loadLedgerAccount(ctx, tx, input.LedgerAccountID)
		if err != nil {
			return err
		}
		if account.ParentID != nil && input.ParentID != nil && *account.ParentID == *input.ParentID {
			return nil
		}
		if hasPostings, err := ledgerAccountSubtreeHasPostings(ctx, tx, account.ID); err != nil {
			return err
		} else if hasPostings {
			return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "a ledger account with postings cannot be moved"}
		}
		height, err := ledgerAccountSubtreeHeight(ctx, tx, account.ID)
		if err != nil {
			return err
		}
		if input.ParentID != nil {
			if err := validateLedgerAccountParent(ctx, tx, &account.ID, *input.ParentID, account.Kind, account.CurrencyID); err != nil {
				return err
			}
			parentDepth, err := ledgerAccountDepth(ctx, tx, *input.ParentID)
			if err != nil {
				return err
			}
			if parentDepth+1+height > 2 {
				return internalerrors.ErrInvalidParameter{Description: "ledger account hierarchy cannot exceed depth 2"}
			}
		} else if height > 2 {
			return internalerrors.ErrInvalidParameter{Description: "ledger account hierarchy cannot exceed depth 2"}
		}
		account.ParentID = input.ParentID
		if _, err := tx.NewUpdate().Model(account).Column("parent_id", "updated_at").WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, "LedgerAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("move ledger account: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
