package accounts

import (
	"context"
	"database/sql"
	"fmt"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// CreateLedgerAccount returns the ledger account with the input's key,
// creating it when there is none (ADR 0005). Under a financial account the key
// is the financial account, the currency, and the name; for the shared
// accounts, which have none, it is the kind, the currency, and the name.
// Names are compared case-insensitively and stored trimmed, in the casing
// they were first given. An account that already
// exists is returned unchanged, closed or not, whatever the rest of the input
// says, so a retry or a second create of the same account is harmless.
func (a *Activities) CreateLedgerAccount(ctx context.Context, input act.CreateLedgerAccountInput) (*act.CreateLedgerAccountResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)
	if err := validateLedgerAccountNameAndCurrency(input.Name, input.CurrencyID); err != nil {
		return nil, err
	}
	if !model.ValidLedgerAccountKind(input.Kind) {
		return nil, internalerrors.ErrInvalidParameter{Description: "invalid ledger account kind"}
	}
	if input.FinancialAccountID != nil && *input.FinancialAccountID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "financial account ID must be positive"}
	}
	if err := refuseOtherCategoryAccount(input); err != nil {
		return nil, err
	}
	if err := refuseReservedName(input); err != nil {
		return nil, err
	}

	existing, err := findLedgerAccount(ctx, db, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &act.CreateLedgerAccountResult{LedgerAccountID: existing.ID}, nil
	}

	result := &act.CreateLedgerAccountResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := ensureCurrencyExists(ctx, tx, input.CurrencyID); err != nil {
			return err
		}
		if input.FinancialAccountID != nil {
			if err := ensureOpenFinancialAccountExists(ctx, tx, *input.FinancialAccountID); err != nil {
				return err
			}
		}

		account := &model.LedgerAccount{
			FinancialAccountID: input.FinancialAccountID,
			CurrencyID:         input.CurrencyID, Name: input.Name, Kind: input.Kind,
		}
		if _, err := tx.NewInsert().Model(account).Returning("id").Exec(ctx); err != nil {
			return fmt.Errorf("insert ledger account: %w", err)
		}
		result.LedgerAccountID = account.ID
		return nil
	})
	if conflict := sqllib.AlreadyExists(err, "LedgerAccount"); conflict != nil {
		// Another create wrote the same key after the lookup above, and the
		// unique index held this insert until it committed. The violation
		// aborted the transaction, so the lookup runs after it.
		existing, err := findLedgerAccount(ctx, db, input)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, conflict
		}
		return &act.CreateLedgerAccountResult{LedgerAccountID: existing.ID}, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// refuseOtherCategoryAccount fails for an income or expense account that is
// not the shared one of its kind. What the money was for is the category of a
// posting, and a second way to say it would split the totals.
func refuseOtherCategoryAccount(input act.CreateLedgerAccountInput) error {
	if !model.IsCategoryKind(input.Kind) {
		return nil
	}
	shared := lt.CategoryAccountName(input.Kind)
	if input.FinancialAccountID == nil && strings.EqualFold(input.Name, shared) {
		return nil
	}
	return internalerrors.ErrInvalidParameter{
		Description: fmt.Sprintf("the only %s account is %q, with no financial account; create a category for %q instead", input.Kind, shared, input.Name),
	}
}

// refuseReservedName fails for an account of no financial account that takes
// the name of a shared or stray account with another kind. The ledger knows
// those accounts by name, and would refuse to rename or close this one as if
// it were its own.
func refuseReservedName(input act.CreateLedgerAccountInput) error {
	kind, reserved := lt.ReservedLedgerAccountKind(input.Name)
	if !reserved || input.FinancialAccountID != nil || input.Kind == kind {
		return nil
	}
	return internalerrors.ErrInvalidParameter{
		Description: fmt.Sprintf("%q is the name of the ledger's %s account and cannot name an %s account with no financial account; pick another name",
			input.Name, kind, input.Kind),
	}
}

// findLedgerAccount returns the ledger account with the input's key, or nil
// when there is none.
func findLedgerAccount(ctx context.Context, db bun.IDB, input act.CreateLedgerAccountInput) (*model.LedgerAccount, error) {
	account := new(model.LedgerAccount)
	query := db.NewSelect().Model(account).
		Where("la.currency_id = ?", input.CurrencyID).
		Where("lower(la.name) = lower(?)", input.Name)
	if input.FinancialAccountID != nil {
		query = query.Where("la.financial_account_id = ?", *input.FinancialAccountID)
	} else {
		query = query.Where("la.financial_account_id IS NULL AND la.kind = ?", input.Kind)
	}

	err := query.Scan(ctx)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("find ledger account: %w", err)
	default:
		return account, nil
	}
}
