package accounts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// deletionTarget is a DeletionTarget as the database has it, which the batch
// activities empty.
type deletionTarget struct {
	// ledgerAccounts are the accounts whose postings a batch handles, by id: the
	// target ledger account, or the financial account's ledger accounts.
	ledgerAccounts map[int64]*model.LedgerAccount
	// financialAccount is the target financial account, whose movements a batch
	// handles too, or nil for a ledger account.
	financialAccount *model.FinancialAccount
	// financialAccounts are the financial accounts the deletion empties, by id:
	// the target one, or in a preview an institution's.
	financialAccounts map[int64]*model.FinancialAccount
	// refusal is why a shared or stray target cannot be emptied: its lines
	// would move onto itself, or onto another shared account.
	refusal error
}

// loadDeletionTargetTx loads the target, or returns nil when it does not
// exist.
func loadDeletionTargetTx(ctx context.Context, tx bun.Tx, target lt.DeletionTarget) (*deletionTarget, error) {
	t := &deletionTarget{ledgerAccounts: map[int64]*model.LedgerAccount{}, financialAccounts: map[int64]*model.FinancialAccount{}}
	if target.LedgerAccountID != nil {
		var account model.LedgerAccount
		err := tx.NewSelect().Model(&account).Where("la.id = ?", *target.LedgerAccountID).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("load ledger account: %w", err)
		}
		t.ledgerAccounts[account.ID] = &account
		if lt.IsReservedLedgerAccount(&account) {
			t.refusal = sharedLedgerAccountConflict(account.Name)
		}
		return t, nil
	}

	var account model.FinancialAccount
	err := tx.NewSelect().Model(&account).Where("fa.id = ?", *target.FinancialAccountID).Scan(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load financial account: %w", err)
	}
	t.financialAccount = &account
	t.financialAccounts[account.ID] = &account

	var ledgerAccounts []model.LedgerAccount
	if err := tx.NewSelect().Model(&ledgerAccounts).Where("la.financial_account_id = ?", account.ID).Scan(ctx); err != nil {
		return nil, fmt.Errorf("load ledger accounts: %w", err)
	}
	for i := range ledgerAccounts {
		t.ledgerAccounts[ledgerAccounts[i].ID] = &ledgerAccounts[i]
	}

	stray, err := isStrayFinancialAccountTx(ctx, tx, &account)
	if err != nil {
		return nil, err
	}
	if stray {
		t.refusal = strayFinancialAccountConflict(account.Name)
	}
	return t, nil
}

// ledgerIDs returns the ids of the target's ledger accounts in ascending
// order.
func (t *deletionTarget) ledgerIDs() []int64 {
	ids := make([]int64, 0, len(t.ledgerAccounts))
	for id := range t.ledgerAccounts {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// isFinancialAccount reports whether id is a financial account the deletion
// empties.
func (t *deletionTarget) isFinancialAccount(id int64) bool {
	_, ok := t.financialAccounts[id]
	return ok
}

// financialIDs returns the ids of the financial accounts the deletion empties
// in ascending order.
func (t *deletionTarget) financialIDs() []int64 {
	ids := make([]int64, 0, len(t.financialAccounts))
	for id := range t.financialAccounts {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

func sharedLedgerAccountConflict(name string) error {
	return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: fmt.Sprintf("%q is shared by the ledger and cannot be deleted while it has postings", name)}
}

func strayFinancialAccountConflict(name string) error {
	return internalerrors.ErrResourceConflict{ResourceType: "FinancialAccount", Reason: fmt.Sprintf("%q holds the lines of deleted accounts and cannot be deleted while it has postings or movements", name)}
}

func foreignChildrenConflict(financialAccountID int64) error {
	return internalerrors.ErrResourceConflict{ResourceType: "FinancialAccount", Reason: fmt.Sprintf("a ledger account of financial account %d has child accounts outside it; delete or move them first", financialAccountID)}
}
