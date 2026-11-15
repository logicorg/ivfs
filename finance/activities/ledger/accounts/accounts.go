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
