package deletion

import (
	"cmp"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// untrackedInstitutionKind is the kind of the "Untracked" institution, which
// stands for no institution in particular.
const untrackedInstitutionKind = "untracked"

// The strays are retrieved or inserted inside the caller's transaction, so a
// unique violation would abort it. The insert skips a conflict instead, after
// waiting for the transaction that wrote the row, and the lookup after it sees
// that row, since each statement of a read committed transaction reads what
// is committed when it starts.
//
// Waiting on another transaction's insert deadlocks when that transaction
// waits on one of ours, so a transaction creates the strays it needs in one
// order: the stray financial account first, then all the stray ledger accounts
// in one call to StrayLedgerAccountsTx, which goes by currency and kind.

// StrayLedgerAccountTx returns the stray ledger account that receives the
// postings, in currencyID, of a deleted ledger account of kind (ADR 0012): a
// balance's or an equity account's go to "Untracked accounts", a category's to
// "Uncategorized" of its kind. It is created top-level, with no financial
// account, the first time it is needed, and returned closed or not, since the
// person cannot close it.
func StrayLedgerAccountTx(ctx context.Context, tx bun.Tx, kind model.LedgerAccountKind, currencyID int64) (int64, error) {
	stray := StrayLedgerAccount(kind, currencyID)
	find := func() (bool, error) {
		err := tx.NewSelect().Model(stray).
			Where("la.parent_id IS NULL AND la.financial_account_id IS NULL").
			Where("la.kind = ? AND la.currency_id = ?", stray.Kind, stray.CurrencyID).
			Where("lower(la.name) = lower(?)", stray.Name).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("find stray ledger account: %w", err)
		}
		return true, nil
	}
	if err := retrieveOrInsert(ctx, tx, stray, find); err != nil {
		return 0, err
	}
	return stray.ID, nil
}

// StrayLedgerAccountsTx returns the stray of each account, by the account's
// id, as StrayLedgerAccountTx does, creating them by currency and kind.
func StrayLedgerAccountsTx(ctx context.Context, tx bun.Tx, accounts []*model.LedgerAccount) (map[int64]int64, error) {
	type key struct {
		currencyID int64
		kind       model.LedgerAccountKind
	}
	keyOf := func(account *model.LedgerAccount) key {
		stray := StrayLedgerAccount(account.Kind, account.CurrencyID)
		return key{stray.CurrencyID, stray.Kind}
	}

	keys := make([]key, 0, len(accounts))
	for _, account := range accounts {
		keys = append(keys, keyOf(account))
	}
	slices.SortFunc(keys, func(a, b key) int {
		return cmp.Or(cmp.Compare(a.currencyID, b.currencyID), cmp.Compare(a.kind, b.kind))
	})

	strayIDs := make(map[key]int64, len(keys))
	for _, k := range slices.Compact(keys) {
		id, err := StrayLedgerAccountTx(ctx, tx, k.kind, k.currencyID)
		if err != nil {
			return nil, err
		}
		strayIDs[k] = id
	}

	byAccount := make(map[int64]int64, len(accounts))
	for _, account := range accounts {
		byAccount[account.ID] = strayIDs[keyOf(account)]
	}
	return byAccount, nil
}

// StrayLedgerAccount is the stray that receives the postings of an account of
// kind in currencyID, before it is retrieved or inserted.
func StrayLedgerAccount(kind model.LedgerAccountKind, currencyID int64) *model.LedgerAccount {
	if kind == model.LedgerAccountExpense || kind == model.LedgerAccountIncome {
		return &model.LedgerAccount{CurrencyID: currencyID, Name: lt.UncategorizedAccountName, Kind: kind}
	}
	return &model.LedgerAccount{CurrencyID: currencyID, Name: lt.UntrackedAccountsName, Kind: model.LedgerAccountEquity}
}

// StrayFinancialAccountTx returns the stray financial account, "Untracked
// accounts" under the "Untracked" institution, which receives the units of a
// deleted financial account (ADR 0012). Both are created the first time they
// are needed, and returned closed or not, since the person cannot close them.
func StrayFinancialAccountTx(ctx context.Context, tx bun.Tx) (int64, error) {
	institution := &model.Institution{Name: lt.UntrackedInstitutionName, Kind: untrackedInstitutionKind}
	findInstitution := func() (bool, error) {
		err := tx.NewSelect().Model(institution).Where("lower(i.name) = lower(?)", institution.Name).Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("find stray institution: %w", err)
		}
		return true, nil
	}
	if err := retrieveOrInsert(ctx, tx, institution, findInstitution); err != nil {
		return 0, err
	}

	stray := &model.FinancialAccount{InstitutionID: institution.ID, Name: lt.UntrackedAccountsName}
	find := func() (bool, error) {
		err := tx.NewSelect().Model(stray).
			Where("fa.institution_id = ?", stray.InstitutionID).
			Where("lower(fa.name) = lower(?)", stray.Name).
			Scan(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		if err != nil {
			return false, fmt.Errorf("find stray financial account: %w", err)
		}
		return true, nil
	}
	if err := retrieveOrInsert(ctx, tx, stray, find); err != nil {
		return 0, err
	}
	return stray.ID, nil
}

// retrieveOrInsert fills row with what find loads, inserting row first when
// find loads nothing.
func retrieveOrInsert(ctx context.Context, tx bun.Tx, row any, find func() (bool, error)) error {
	if found, err := find(); found || err != nil {
		return err
	}
	if _, err := tx.NewInsert().Model(row).On("CONFLICT DO NOTHING").Exec(ctx); err != nil {
		return fmt.Errorf("insert stray account: %w", err)
	}
	found, err := find()
	if err == nil && !found {
		return fmt.Errorf("insert stray account: %T vanished after its insert", row)
	}
	return err
}
