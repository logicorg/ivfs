package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// CreateLedgerAccount returns the ledger account with the input's key,
// creating it when there is none (ADR 0005). Under a financial account the key
// is the financial account, the currency, and the name; elsewhere, for
// categories, equity, and clearing accounts, it is the parent, the kind, the
// currency, and the name. Names are compared case-insensitively and stored
// trimmed, in the casing they were first given. An account that already
// exists is returned unchanged, closed or not, whatever the rest of the input
// says, so a retry or a second create of the same account is harmless.
func (a *Activities) CreateLedgerAccount(ctx context.Context, input lt.CreateLedgerAccountInput) (*lt.CreateLedgerAccountResult, error) {
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

	existing, err := findLedgerAccount(ctx, db, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &lt.CreateLedgerAccountResult{LedgerAccountID: existing.ID}, nil
	}

	result := &lt.CreateLedgerAccountResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if err := ensureCurrencyExists(ctx, tx, input.CurrencyID); err != nil {
			return err
		}
		if input.FinancialAccountID != nil {
			if err := ensureOpenFinancialAccountExists(ctx, tx, *input.FinancialAccountID); err != nil {
				return err
			}
		}
		if input.ParentID != nil {
			if err := validateLedgerAccountParent(ctx, tx, nil, *input.ParentID, input.Kind, input.CurrencyID); err != nil {
				return err
			}
		}

		account := &model.LedgerAccount{
			ParentID: input.ParentID, FinancialAccountID: input.FinancialAccountID,
			CurrencyID: input.CurrencyID, Name: input.Name, Kind: input.Kind,
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
		return &lt.CreateLedgerAccountResult{LedgerAccountID: existing.ID}, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// findLedgerAccount returns the ledger account with the input's key, or nil
// when there is none. A missing parent matches only a top-level account.
func findLedgerAccount(ctx context.Context, db bun.IDB, input lt.CreateLedgerAccountInput) (*model.LedgerAccount, error) {
	account := new(model.LedgerAccount)
	query := db.NewSelect().Model(account).
		Where("la.currency_id = ?", input.CurrencyID).
		Where("lower(la.name) = lower(?)", input.Name)
	switch {
	case input.FinancialAccountID != nil:
		query = query.Where("la.financial_account_id = ?", *input.FinancialAccountID)
	case input.ParentID != nil:
		query = query.Where("la.financial_account_id IS NULL AND la.kind = ? AND la.parent_id = ?", input.Kind, *input.ParentID)
	default:
		query = query.Where("la.financial_account_id IS NULL AND la.kind = ? AND la.parent_id IS NULL", input.Kind)
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
