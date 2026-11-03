package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type Activities struct{ Pools tenancy.Pools }

func validateLedgerAccountNameAndCurrency(name string, currencyID int64) error {
	if strings.TrimSpace(name) == "" {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}
	if currencyID <= 0 {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"CurrencyID"}}
	}
	return nil
}

func ensureCurrencyExists(ctx context.Context, db bun.Tx, currencyID int64) error {
	exists, err := model.FindCurrencyByIDQuery(db, currencyID).Exists(ctx)
	if err != nil {
		return fmt.Errorf("check currency: %w", err)
	}
	if !exists {
		return internalerrors.ErrNotFound{ResourceType: "Currency", ID: currencyID}
	}
	return nil
}

func ensureOpenFinancialAccountExists(ctx context.Context, db bun.Tx, accountID int64) error {
	account := new(model.FinancialAccount)

	err := model.FindFinancialAccountByIDQuery(db, accountID).Scan(ctx, account)
	if err != nil {
		if err == sql.ErrNoRows {
			return internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: accountID}
		}
		return fmt.Errorf("check financial account: %w", err)
	}
	if account.ClosedAt != nil {
		return internalerrors.ErrResourceClosed{ResourceType: "FinancialAccount", ID: accountID}
	}
	return nil
}

func loadLedgerAccount(ctx context.Context, db bun.Tx, accountID int64) (*model.LedgerAccount, error) {
	if accountID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "ledger account ID must be positive"}
	}

	account := new(model.LedgerAccount)

	err := model.FindLedgerAccountByIDQuery(db, accountID).Scan(ctx, account)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: accountID}
		}
		return nil, fmt.Errorf("load ledger account: %w", err)
	}
	return account, nil
}

func validateLedgerAccountParent(ctx context.Context, db bun.Tx, accountID *int64, parentID int64, kind model.LedgerAccountKind, currencyID int64) error {
	if parentID <= 0 {
		return internalerrors.ErrInvalidParameter{Description: "parent ledger account ID must be positive"}
	}
	if accountID != nil && *accountID == parentID {
		return internalerrors.ErrInvalidParameter{Description: "ledger account cannot be its own parent"}
	}

	parent, err := loadLedgerAccount(ctx, db, parentID)
	if err != nil {
		return err
	}
	if parent.ClosedAt != nil {
		return internalerrors.ErrResourceClosed{ResourceType: "LedgerAccount", ID: parentID, Reason: fmt.Sprintf("parent ledger account %d is closed", parentID)}
	}
	if parent.Kind != kind {
		return internalerrors.ErrInvalidParameter{Description: "parent ledger account must have the same kind"}
	}
	if parent.CurrencyID != currencyID {
		return internalerrors.ErrInvalidParameter{Description: "parent ledger account must have the same currency"}
	}
	if hasDirectPostings, err := ledgerAccountHasPostings(ctx, db, parent.ID); err != nil {
		return err
	} else if hasDirectPostings {
		return internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "a ledger account with postings cannot become a grouping account"}
	}

	depth, err := ledgerAccountDepth(ctx, db, parent.ID)
	if err != nil {
		return err
	}
	if depth+1 > 2 {
		return internalerrors.ErrInvalidParameter{Description: "ledger account hierarchy cannot exceed depth 2"}
	}

	if accountID != nil {
		if descendant, err := ledgerAccountIsDescendant(ctx, db, parent.ID, *accountID); err != nil {
			return err
		} else if descendant {
			return internalerrors.ErrInvalidParameter{Description: "ledger account move would create a cycle"}
		}
	}
	return nil
}

func ledgerAccountDepth(ctx context.Context, db bun.Tx, accountID int64) (int, error) {
	depth := 0
	seen := map[int64]struct{}{}
	currentID := accountID
	for {
		if _, ok := seen[currentID]; ok {
			return 0, internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account hierarchy contains a cycle"}
		}
		seen[currentID] = struct{}{}

		account, err := loadLedgerAccount(ctx, db, currentID)
		if err != nil {
			return 0, err
		}
		if account.ParentID == nil {
			return depth, nil
		}
		depth++
		currentID = *account.ParentID
		if depth > 2 {
			return depth, nil
		}
	}
}

func ledgerAccountHasPostings(ctx context.Context, db bun.Tx, accountID int64) (bool, error) {
	exists, err := db.NewSelect().Model((*model.Posting)(nil)).Where("ledger_account_id = ?", accountID).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check ledger account postings: %w", err)
	}
	return exists, nil
}

// refuseStrandedBalance refuses to close an asset or liability ledger account
// that still has a balance, which closing would strand: it must be moved out
// first. The balances of income, expense and equity accounts are history, so
// they close freely. The account stays locked until the transaction ends, so a
// posting cannot change the balance before it is closed.
func refuseStrandedBalance(ctx context.Context, db bun.Tx, account *model.LedgerAccount) error {
	if account.Kind != model.LedgerAccountAsset && account.Kind != model.LedgerAccountLiability {
		return nil
	}
	if _, err := db.NewSelect().Model((*model.LedgerAccount)(nil)).Column("la.id").
		Where("la.id = ?", account.ID).For("NO KEY UPDATE").Exec(ctx); err != nil {
		return fmt.Errorf("lock ledger account: %w", err)
	}

	var balance decimal.Decimal
	err := db.NewRaw(`
		SELECT COALESCE(SUM(CASE WHEN p.side = 'debit' THEN p.amount ELSE -p.amount END), 0)
		FROM ledger.posting AS p
		JOIN ledger.journal_entry AS je ON je.id = p.journal_entry_id
		WHERE p.ledger_account_id = ?
		  AND je.status = 'posted'`, account.ID).Scan(ctx, &balance)
	if err != nil {
		return fmt.Errorf("calculate ledger account balance: %w", err)
	}
	if account.Kind == model.LedgerAccountLiability {
		balance = balance.Neg()
	}
	if !balance.IsZero() {
		return internalerrors.ErrResourceConflict{
			ResourceType: "LedgerAccount",
			Reason:       fmt.Sprintf("ledger account still has a balance of %s", balance),
		}
	}
	return nil
}

func ledgerAccountSubtreeHasPostings(ctx context.Context, db bun.Tx, accountID int64) (bool, error) {
	ids, err := ledgerAccountSubtreeIDs(ctx, db, accountID)
	if err != nil {
		return false, err
	}
	exists, err := db.NewSelect().Model((*model.Posting)(nil)).Where("ledger_account_id IN (?)", bun.In(ids)).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check ledger account subtree postings: %w", err)
	}
	return exists, nil
}

func ledgerAccountSubtreeIDs(ctx context.Context, db bun.Tx, accountID int64) ([]int64, error) {
	var accounts []model.LedgerAccount
	if err := db.NewSelect().Model(&accounts).Column("id", "parent_id").Scan(ctx); err != nil {
		return nil, fmt.Errorf("load ledger account hierarchy: %w", err)
	}
	children := make(map[int64][]int64, len(accounts))
	for _, account := range accounts {
		if account.ParentID != nil {
			children[*account.ParentID] = append(children[*account.ParentID], account.ID)
		}
	}

	ids := []int64{accountID}
	seen := map[int64]struct{}{accountID: {}}
	for index := 0; index < len(ids); index++ {
		for _, childID := range children[ids[index]] {
			if _, ok := seen[childID]; ok {
				return nil, internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account hierarchy contains a cycle"}
			}
			seen[childID] = struct{}{}
			ids = append(ids, childID)
		}
	}
	return ids, nil
}

func ledgerAccountSubtreeHeight(ctx context.Context, db bun.Tx, accountID int64) (int, error) {
	var accounts []model.LedgerAccount
	if err := db.NewSelect().Model(&accounts).Column("id", "parent_id").Scan(ctx); err != nil {
		return 0, fmt.Errorf("load ledger account hierarchy: %w", err)
	}
	children := make(map[int64][]int64, len(accounts))
	for _, account := range accounts {
		if account.ParentID != nil {
			children[*account.ParentID] = append(children[*account.ParentID], account.ID)
		}
	}
	type node struct {
		id    int64
		depth int
	}
	queue := []node{{id: accountID}}
	seen := map[int64]struct{}{accountID: {}}
	height := 0
	for index := 0; index < len(queue); index++ {
		current := queue[index]
		if current.depth > height {
			height = current.depth
		}
		for _, childID := range children[current.id] {
			if _, ok := seen[childID]; ok {
				return 0, internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account hierarchy contains a cycle"}
			}
			seen[childID] = struct{}{}
			queue = append(queue, node{id: childID, depth: current.depth + 1})
		}
	}
	return height, nil
}

func ledgerAccountIsDescendant(ctx context.Context, db bun.Tx, candidateParentID, accountID int64) (bool, error) {
	currentID := candidateParentID
	seen := map[int64]struct{}{}
	for {
		if currentID == accountID {
			return true, nil
		}
		if _, ok := seen[currentID]; ok {
			return false, internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account hierarchy contains a cycle"}
		}
		seen[currentID] = struct{}{}
		account, err := loadLedgerAccount(ctx, db, currentID)
		if err != nil {
			return false, err
		}
		if account.ParentID == nil {
			return false, nil
		}
		currentID = *account.ParentID
	}
}
