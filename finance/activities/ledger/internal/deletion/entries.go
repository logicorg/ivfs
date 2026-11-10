// Package deletion holds the steps shared by the activities that delete
// recorded history: a journal entry or a movement group (ADR 0011), and the
// batches that empty an account being deleted (ADR 0012).
package deletion

import (
	"context"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/positions"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// LockAccountsOfTx locks the accounts the entries touch, as LockAccountsTx
// does.
func LockAccountsOfTx(ctx context.Context, tx bun.Tx, entryIDs []int64) error {
	financialIDs, ledgerIDs, err := AccountsOfTx(ctx, tx, entryIDs)
	if err != nil {
		return err
	}
	return LockAccountsTx(ctx, tx, financialIDs, ledgerIDs)
}

// AccountsOfTx returns the financial accounts the entries move units in, and
// the ledger accounts they post to.
func AccountsOfTx(ctx context.Context, tx bun.Tx, entryIDs []int64) (financialIDs, ledgerIDs []int64, err error) {
	var movements []model.PositionMovement
	if err := tx.NewSelect().Model(&movements).Where("pm.journal_entry_id IN (?)", bun.List(entryIDs)).Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("load position movements: %w", err)
	}
	var postings []model.Posting
	if err := tx.NewSelect().Model(&postings).Where("journal_entry_id IN (?)", bun.List(entryIDs)).Scan(ctx); err != nil {
		return nil, nil, fmt.Errorf("load journal postings: %w", err)
	}
	for _, movement := range movements {
		financialIDs = append(financialIDs, movement.FinancialAccountID)
	}
	for _, posting := range postings {
		ledgerIDs = append(ledgerIDs, posting.LedgerAccountID)
	}
	return financialIDs, ledgerIDs, nil
}

// LockAccountsTx locks the ledger accounts and the financial accounts, with
// those the ledger accounts belong to, in the order posting takes them: the
// financial accounts and then the ledger accounts, each in ascending id order.
func LockAccountsTx(ctx context.Context, tx bun.Tx, financialIDs, ledgerIDs []int64) error {
	if len(ledgerIDs) > 0 {
		var ledgerAccounts []model.LedgerAccount
		if err := tx.NewSelect().Model(&ledgerAccounts).Where("la.id IN (?)", bun.List(ledgerIDs)).Scan(ctx); err != nil {
			return fmt.Errorf("load ledger accounts: %w", err)
		}
		for _, account := range ledgerAccounts {
			if account.FinancialAccountID != nil {
				financialIDs = append(financialIDs, *account.FinancialAccountID)
			}
		}
	}
	if _, err := positions.LockFinancialAccountsTx(ctx, tx, financialIDs); err != nil {
		return err
	}

	if len(ledgerIDs) == 0 {
		return nil
	}
	// The row lock conflicts with the share lock that posting takes, so a posting
	// or a close waits for this transaction.
	var locked []model.LedgerAccount
	if err := tx.NewSelect().Model(&locked).Column("id").Where("la.id IN (?)", bun.List(ledgerIDs)).OrderExpr("la.id").For("NO KEY UPDATE").Scan(ctx); err != nil {
		return fmt.Errorf("lock ledger accounts: %w", err)
	}
	return nil
}

// DeleteEntryTx deletes an entry with its movements and postings, declaring it
// to the triggers that guard them. The setting ends with the transaction.
func DeleteEntryTx(ctx context.Context, tx bun.Tx, entryID int64) error {
	if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_journal_entry', ?, true)`, fmt.Sprint(entryID)); err != nil {
		return fmt.Errorf("declare journal entry deletion: %w", err)
	}
	if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("journal_entry_id = ?", entryID).Exec(ctx); err != nil {
		return fmt.Errorf("delete position movements: %w", err)
	}
	if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", entryID).Exec(ctx); err != nil {
		return fmt.Errorf("delete journal postings: %w", err)
	}
	if _, err := tx.NewDelete().Model((*model.JournalEntry)(nil)).Where("id = ?", entryID).Exec(ctx); err != nil {
		return fmt.Errorf("delete journal entry: %w", err)
	}
	return nil
}
