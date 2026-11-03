package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// DeleteFinancialAccount deletes the rows of a financial account emptied by a
// deletion (ADR 0012): its ledger accounts, children first, and then the
// account itself. An account that does not exist is already deleted.
//
// A row something still refers to, such as a posting or a movement recorded
// since the batches emptied it, fails with ErrResourceConflict and deletes
// nothing, so the deletion can empty it again. So does a ledger account with a
// child outside the financial account.
func (a *Activities) DeleteFinancialAccount(ctx context.Context, input lt.DeleteFinancialAccountInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// The financial account before its ledger accounts, in the order posting
		// locks them.
		err := tx.NewSelect().Model((*model.FinancialAccount)(nil)).Column("fa.id").
			Where("fa.id = ?", input.FinancialAccountID).For("UPDATE").
			Scan(ctx, new(int64))
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("lock financial account: %w", err)
		}

		if err := deleteLedgerAccountsOfTx(ctx, tx, input.FinancialAccountID); err != nil {
			return err
		}

		if _, err := tx.NewDelete().Model((*model.FinancialAccount)(nil)).Where("id = ?", input.FinancialAccountID).Exec(ctx); err != nil {
			if conflict := sqllib.StillReferenced(err, "FinancialAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("delete financial account: %w", err)
		}
		return nil
	})
}

// deleteLedgerAccountsOfTx deletes the financial account's ledger accounts
// leaf by leaf, since every key restricts deletion row by row, so a parent
// deleted in the same statement as its child would still be referenced.
func deleteLedgerAccountsOfTx(ctx context.Context, tx bun.Tx, financialAccountID int64) error {
	for {
		result, err := tx.NewDelete().Model((*model.LedgerAccount)(nil)).
			Where("la.financial_account_id = ?", financialAccountID).
			Where("NOT EXISTS (?)", tx.NewSelect().Model((*model.LedgerAccount)(nil)).ModelTableExpr("ledger.ledger_account AS child").
				ColumnExpr("1").Where("child.parent_id = la.id")).
			Exec(ctx)
		if err != nil {
			if conflict := sqllib.StillReferenced(err, "LedgerAccount"); conflict != nil {
				return conflict
			}
			return fmt.Errorf("delete ledger accounts: %w", err)
		}
		deleted, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("count deleted ledger accounts: %w", err)
		}
		if deleted > 0 {
			continue
		}

		// No leaf is left, so what is left has a child elsewhere.
		left, err := tx.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.financial_account_id = ?", financialAccountID).Exists(ctx)
		if err != nil {
			return fmt.Errorf("check ledger accounts: %w", err)
		}
		if left {
			return foreignChildrenConflict(financialAccountID)
		}
		return nil
	}
}
