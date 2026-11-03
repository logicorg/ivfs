package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// CloseAccountForDeletion closes the target of a deletion, so nothing new is
// recorded on it while the deletion empties it (ADR 0012). Unlike
// CloseLedgerAccount and CloseFinancialAccount, it closes an account with a
// balance or units, since the deletion moves them off, and a financial
// account's ledger accounts with it. A target that does not exist is reported
// gone. It is idempotent: an account already closed keeps its ClosedAt.
//
// It refuses, with ErrResourceConflict, an account with child ledger accounts,
// whose rows could not be deleted after it, and a shared or stray account that
// has postings or movements.
func (a *Activities) CloseAccountForDeletion(ctx context.Context, target lt.DeletionTarget) (*lt.CloseAccountForDeletionResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := target.Validate(); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	result := &lt.CloseAccountForDeletionResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		var err error
		if target.LedgerAccountID != nil {
			result.Gone, err = closeLedgerAccountForDeletionTx(ctx, tx, *target.LedgerAccountID, now)
		} else {
			result.Gone, err = closeFinancialAccountForDeletionTx(ctx, tx, *target.FinancialAccountID, now)
		}
		return err
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func closeLedgerAccountForDeletionTx(ctx context.Context, tx bun.Tx, id int64, now time.Time) (bool, error) {
	// Locking the row makes a posting in flight finish first, and a later one
	// find the account closed, so the checks below hold until the close.
	var account model.LedgerAccount
	err := tx.NewSelect().Model(&account).Where("la.id = ?", id).For("NO KEY UPDATE").Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock ledger account: %w", err)
	}

	if err := refuseChildAccountsTx(ctx, tx, id); err != nil {
		return false, err
	}

	if lt.IsReservedLedgerAccount(&account) {
		hasPostings, err := ledgerAccountHasPostings(ctx, tx, id)
		if err != nil {
			return false, err
		}
		if hasPostings {
			return false, sharedLedgerAccountConflict(account.Name)
		}
	}

	return false, closeOpenTx(ctx, tx, (*model.LedgerAccount)(nil), "la.id = ?", id, now)
}

func closeFinancialAccountForDeletionTx(ctx context.Context, tx bun.Tx, id int64, now time.Time) (bool, error) {
	// The financial account and then its ledger accounts, in the order posting
	// locks them.
	locked, err := positions.LockFinancialAccountsTx(ctx, tx, []int64{id})

	if notFound := new(internalerrors.ErrNotFound); errors.As(err, notFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	account := locked[id]

	var ledgerIDs []int64
	if err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).Column("la.id").
		Where("la.financial_account_id = ?", id).OrderExpr("la.id").For("NO KEY UPDATE").
		Scan(ctx, &ledgerIDs); err != nil {
		return false, fmt.Errorf("lock ledger accounts: %w", err)
	}

	if err := refuseForeignChildrenTx(ctx, tx, id, ledgerIDs); err != nil {
		return false, err
	}

	stray, err := isStrayFinancialAccountTx(ctx, tx, account)
	if err != nil {
		return false, err
	}
	if stray {
		used, err := financialAccountUsedTx(ctx, tx, id, ledgerIDs)
		if err != nil {
			return false, err
		}
		if used {
			return false, strayFinancialAccountConflict(account.Name)
		}
	}

	if err := closeOpenTx(ctx, tx, (*model.FinancialAccount)(nil), "fa.id = ?", id, now); err != nil {
		return false, err
	}
	if len(ledgerIDs) == 0 {
		return false, nil
	}
	return false, closeOpenTx(ctx, tx, (*model.LedgerAccount)(nil), "la.id IN (?)", bun.List(ledgerIDs), now)
}

// refuseChildAccountsTx refuses a ledger account with child accounts, whose
// rows could not be deleted after it.
func refuseChildAccountsTx(ctx context.Context, tx bun.Tx, id int64) error {
	children, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.parent_id = ?", id).Exists(ctx)
	if err != nil {
		return fmt.Errorf("check child ledger accounts: %w", err)
	}
	if children {
		return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: fmt.Sprintf("ledger account %d has child accounts; delete them first", id)}
	}
	return nil
}

// refuseForeignChildrenTx refuses a financial account whose ledger accounts,
// ledgerIDs, have children outside it. Its children of its own are deleted
// with them.
func refuseForeignChildrenTx(ctx context.Context, tx bun.Tx, id int64, ledgerIDs []int64) error {
	if len(ledgerIDs) == 0 {
		return nil
	}
	foreignChildren, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).
		Where("la.parent_id IN (?)", bun.List(ledgerIDs)).
		Where("la.financial_account_id IS DISTINCT FROM ?", id).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("check child ledger accounts: %w", err)
	}
	if foreignChildren {
		return foreignChildrenConflict(id)
	}
	return nil
}

// isStrayFinancialAccountTx reports whether the account is "Untracked
// accounts" under the "Untracked" institution.
func isStrayFinancialAccountTx(ctx context.Context, tx bun.Tx, account *model.FinancialAccount) (bool, error) {
	if !lt.IsUntrackedFinancialAccountName(account.Name) {
		return false, nil
	}
	var institution model.Institution
	if err := tx.NewSelect().Model(&institution).Column("i.name").Where("i.id = ?", account.InstitutionID).Scan(ctx); err != nil {
		return false, fmt.Errorf("load institution: %w", err)
	}
	return lt.IsUntrackedInstitution(institution.Name), nil
}

func financialAccountUsedTx(ctx context.Context, tx bun.Tx, id int64, ledgerIDs []int64) (bool, error) {
	movements, err := tx.NewSelect().Model((*model.PositionMovement)(nil)).Where("pm.financial_account_id = ?", id).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check position movements: %w", err)
	}
	if movements || len(ledgerIDs) == 0 {
		return movements, nil
	}
	postings, err := tx.NewSelect().Model((*model.Posting)(nil)).Where("ledger_account_id IN (?)", bun.List(ledgerIDs)).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check postings: %w", err)
	}
	return postings, nil
}

// closeOpenTx closes the rows of table that match where and are still open,
// so an account closed earlier keeps its ClosedAt.
func closeOpenTx(ctx context.Context, tx bun.Tx, table any, where string, arg any, now time.Time) error {
	if _, err := tx.NewUpdate().Model(table).
		Set("closed_at = ?", now).Set("updated_at = ?", now).
		Where(where, arg).Where("closed_at IS NULL").
		Exec(ctx); err != nil {
		return fmt.Errorf("close %T: %w", table, err)
	}
	return nil
}
