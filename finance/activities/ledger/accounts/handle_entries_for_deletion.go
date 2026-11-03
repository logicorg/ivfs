package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// HandleEntriesForDeletion handles the next batch of journal entries on the
// target of a deletion, up to input.Limit in id order, in one transaction
// (ADR 0012). A posted entry that also touches a surviving balance, or units on
// a surviving financial account other than the stray one, is kept, and its
// lines on the target move to the stray accounts. Any other entry is deleted
// with its reversals, as DeleteJournalEntry does. Either way it no longer
// touches the target, so a batch that handles nothing means none are left, and
// a batch run again moves nothing twice.
//
// It refuses a shared or stray target that has entries with
// ErrResourceConflict. A target that does not exist has nothing to handle.
func (a *Activities) HandleEntriesForDeletion(ctx context.Context, input lt.DeletionBatchInput) (*lt.DeletionBatchResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	result := &lt.DeletionBatchResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		target, err := loadDeletionTargetTx(ctx, tx, input.Target)
		if err != nil || target == nil {
			return err
		}
		entryIDs, err := target.entriesTx(ctx, tx, 0, input.Limit)
		if err != nil || len(entryIDs) == 0 {
			return err
		}
		if target.refusal != nil {
			return target.refusal
		}

		financialIDs, ledgerIDs, err := deletion.AccountsOfTx(ctx, tx, entryIDs)
		if err != nil {
			return err
		}
		if target.financialAccount != nil {
			financialIDs = append(financialIDs, target.financialAccount.ID)
		}
		if err := deletion.LockAccountsTx(ctx, tx, financialIDs, append(ledgerIDs, target.ledgerIDs()...)); err != nil {
			return err
		}

		var kept []*model.JournalEntry
		for _, id := range entryIDs {
			entry, err := loadEntryForDeletionTx(ctx, tx, id)
			if err != nil {
				return err
			}
			// Gone with the chain of an entry before it in the batch, or deleted
			// meanwhile.
			if entry == nil {
				continue
			}
			keep, err := target.keepsTx(ctx, tx, entry)
			if err != nil {
				return err
			}
			if keep {
				kept = append(kept, entry)
				continue
			}
			if err := deleteChainTx(ctx, tx, id); err != nil {
				return err
			}
		}
		if err := target.moveLinesTx(ctx, tx, kept); err != nil {
			return err
		}
		result.Handled = len(entryIDs)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// entriesTx returns the ids of up to limit entries after the id after that
// post to the target's ledger accounts or move units in its financial
// accounts, in ascending order.
func (t *deletionTarget) entriesTx(ctx context.Context, tx bun.Tx, after int64, limit int) ([]int64, error) {
	// Without accounts the filter below would match every entry.
	if len(t.ledgerAccounts) == 0 && len(t.financialAccounts) == 0 {
		return nil, nil
	}
	var ids []int64
	err := tx.NewSelect().Model((*model.JournalEntry)(nil)).Column("je.id").
		WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			if ledgerIDs := t.ledgerIDs(); len(ledgerIDs) > 0 {
				q = q.WhereOr("EXISTS (?)", tx.NewSelect().Model((*model.Posting)(nil)).ColumnExpr("1").
					Where("p.journal_entry_id = je.id").Where("p.ledger_account_id IN (?)", bun.List(ledgerIDs)))
			}
			if financialIDs := t.financialIDs(); len(financialIDs) > 0 {
				q = q.WhereOr("EXISTS (?)", tx.NewSelect().Model((*model.PositionMovement)(nil)).ColumnExpr("1").
					Where("pm.journal_entry_id = je.id").Where("pm.financial_account_id IN (?)", bun.List(financialIDs)))
			}
			return q
		}).
		Where("je.id > ?", after).
		OrderExpr("je.id").Limit(limit).
		Scan(ctx, &ids)
	if err != nil {
		return nil, fmt.Errorf("list journal entries on the deleted account: %w", err)
	}
	return ids, nil
}

// loadEntryForDeletionTx loads an entry with its postings, their accounts, and
// its movements, or returns nil when it does not exist.
func loadEntryForDeletionTx(ctx context.Context, tx bun.Tx, id int64) (*model.JournalEntry, error) {
	var entry model.JournalEntry
	err := tx.NewSelect().Model(&entry).
		Relation("Postings").Relation("Postings.LedgerAccount").
		Relation("PositionMovements").Relation("PositionMovements.FinancialAccount").
		Where("je.id = ?", id).
		Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load journal entry: %w", err)
	}
	return &entry, nil
}

// keepsTx reports whether the entry survives the deletion: it is posted, and
// changes a balance or holdings that survive. "FX clearing" and the strays are
// not balances, and the stray financial account's holdings are not tracked.
func (t *deletionTarget) keepsTx(ctx context.Context, tx bun.Tx, entry *model.JournalEntry) (bool, error) {
	if entry.Status != model.JournalEntryPosted {
		return false, nil
	}
	for _, posting := range entry.Postings {
		if _, deleted := t.ledgerAccounts[posting.LedgerAccountID]; deleted {
			continue
		}
		account := posting.LedgerAccount
		if (account.Kind == model.LedgerAccountAsset || account.Kind == model.LedgerAccountLiability) && !lt.IsReservedLedgerAccount(account) {
			return true, nil
		}
	}
	for _, movement := range entry.PositionMovements {
		if t.isFinancialAccount(movement.FinancialAccountID) {
			continue
		}
		stray, err := isStrayFinancialAccountTx(ctx, tx, movement.FinancialAccount)
		if err != nil {
			return false, err
		}
		if !stray {
			return true, nil
		}
	}
	return false, nil
}

// deleteChainTx deletes the entry with the chain of reversals that follow it,
// the last one first, since a reversal refers to what it reverses.
func deleteChainTx(ctx context.Context, tx bun.Tx, entryID int64) error {
	chain, err := deletion.ReversalChainTx(ctx, tx, entryID)
	if err != nil {
		return err
	}
	for n := len(chain) - 1; n >= 0; n-- {
		if err := deletion.DeleteEntryTx(ctx, tx, chain[n]); err != nil {
			return err
		}
	}
	return nil
}

// moveLinesTx moves the lines the kept entries have on the target to the
// strays: each posting to the stray of its account's kind and currency, and
// each movement to the stray financial account. The triggers let a posted line
// move only off the account the transaction declares, so it declares each
// account in turn.
func (t *deletionTarget) moveLinesTx(ctx context.Context, tx bun.Tx, kept []*model.JournalEntry) error {
	if len(kept) == 0 {
		return nil
	}

	entryIDs := make([]int64, 0, len(kept))
	moving := map[int64]bool{}
	movesUnits := false
	for _, entry := range kept {
		entryIDs = append(entryIDs, entry.ID)
		for _, posting := range entry.Postings {
			if _, deleted := t.ledgerAccounts[posting.LedgerAccountID]; deleted {
				moving[posting.LedgerAccountID] = true
			}
		}
		for _, movement := range entry.PositionMovements {
			movesUnits = movesUnits || t.isFinancialAccount(movement.FinancialAccountID)
		}
	}

	var strayFinancialID int64
	if movesUnits {
		var err error
		if strayFinancialID, err = deletion.StrayFinancialAccountTx(ctx, tx); err != nil {
			return err
		}
	}
	var accounts []*model.LedgerAccount
	for _, id := range t.ledgerIDs() {
		if moving[id] {
			accounts = append(accounts, t.ledgerAccounts[id])
		}
	}
	strays, err := deletion.StrayLedgerAccountsTx(ctx, tx, accounts)
	if err != nil {
		return err
	}

	for _, account := range accounts {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_ledger_account', ?, true)`, fmt.Sprint(account.ID)); err != nil {
			return fmt.Errorf("declare ledger account deletion: %w", err)
		}
		if _, err := tx.NewUpdate().Model((*model.Posting)(nil)).
			Set("ledger_account_id = ?", strays[account.ID]).
			Where("p.ledger_account_id = ?", account.ID).Where("p.journal_entry_id IN (?)", bun.List(entryIDs)).
			Exec(ctx); err != nil {
			return fmt.Errorf("move journal postings: %w", err)
		}
	}

	if !movesUnits {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_financial_account', ?, true)`, fmt.Sprint(t.financialAccount.ID)); err != nil {
		return fmt.Errorf("declare financial account deletion: %w", err)
	}
	if _, err := tx.NewUpdate().Model((*model.PositionMovement)(nil)).
		Set("financial_account_id = ?", strayFinancialID).
		Where("pm.financial_account_id = ?", t.financialAccount.ID).Where("pm.journal_entry_id IN (?)", bun.List(entryIDs)).
		Exec(ctx); err != nil {
		return fmt.Errorf("move position movements: %w", err)
	}
	return nil
}
