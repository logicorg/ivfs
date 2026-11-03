package accounts

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// cashLedgerAccountName is the name of the cash ledger account created with a
// financial account. The financial account and the currency already identify
// it, so the name does not repeat them (ADR 0005).
const cashLedgerAccountName = "Cash"

// CreateFinancialAccount returns the financial account with the given name at
// the given institution, compared case-insensitively, creating it when there
// is none (ADR 0005). An account that already exists is returned unchanged,
// closed or not, whatever the rest of the input says, so a retry or a second
// create of the same account is harmless.
//
// When a cash currency is given, a new account is created together with a
// "Cash" ledger account in that currency. For an existing account, the result
// carries its "Cash" ledger account in that currency if it has one, so that a
// retry returns exactly what the first call did; none is created.
func (a *Activities) CreateFinancialAccount(ctx context.Context, input lt.CreateFinancialAccountInput) (*lt.CreateFinancialAccountResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(input.Name)

	existing, err := findFinancialAccount(ctx, db, input.InstitutionID, name, input.CashCurrencyID)
	if err != nil || existing != nil {
		return existing, err
	}

	result := &lt.CreateFinancialAccountResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		account := &model.FinancialAccount{InstitutionID: input.InstitutionID, Name: name, ExternalID: input.ExternalID}
		if _, err := tx.NewInsert().Model(account).Returning("id").Exec(ctx); err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return notFound
			}
			return fmt.Errorf("insert financial account: %w", err)
		}
		result.FinancialAccountID = account.ID

		if input.CashCurrencyID == nil {
			return nil
		}

		cash := &model.LedgerAccount{
			FinancialAccountID: &account.ID,
			CurrencyID:         *input.CashCurrencyID,
			Name:               cashLedgerAccountName,
			Kind:               model.LedgerAccountAsset,
		}
		if _, err := tx.NewInsert().Model(cash).Returning("id").Exec(ctx); err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return notFound
			}
			return fmt.Errorf("insert financial account cash ledger account: %w", err)
		}
		result.CashLedgerAccountID = &cash.ID
		return nil
	})
	if conflict := sqllib.AlreadyExists(err, "FinancialAccount"); conflict != nil {
		// The unique violation aborted the transaction, so the lookup runs after
		// it. Either another create wrote the same name after the lookup above,
		// and the index held this insert until it committed, or the external ID
		// is another account's, which no lookup by name finds.
		existing, err := findFinancialAccount(ctx, db, input.InstitutionID, name, input.CashCurrencyID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, conflict
		}
		return existing, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// findFinancialAccount returns the result of creating the financial account
// with the given name at the institution, compared case-insensitively, if it
// already exists, together with its "Cash" ledger account in cashCurrencyID
// when one is asked for and exists. It returns nil when there is no such
// account.
func findFinancialAccount(ctx context.Context, db bun.IDB, institutionID int64, name string, cashCurrencyID *int64) (*lt.CreateFinancialAccountResult, error) {
	account := new(model.FinancialAccount)
	err := db.NewSelect().Model(account).
		Where("fa.institution_id = ?", institutionID).
		Where("lower(fa.name) = lower(?)", name).
		Scan(ctx)
	switch {
	case err == sql.ErrNoRows:
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("find financial account: %w", err)
	}

	result := &lt.CreateFinancialAccountResult{FinancialAccountID: account.ID}
	if cashCurrencyID == nil {
		return result, nil
	}

	cash := new(model.LedgerAccount)
	err = db.NewSelect().Model(cash).
		Where("la.financial_account_id = ?", account.ID).
		Where("la.currency_id = ?", *cashCurrencyID).
		Where("lower(la.name) = lower(?)", cashLedgerAccountName).
		Scan(ctx)
	switch {
	case err == sql.ErrNoRows:
		return result, nil
	case err != nil:
		return nil, fmt.Errorf("find financial account cash ledger account: %w", err)
	}
	result.CashLedgerAccountID = &cash.ID
	return result, nil
}
