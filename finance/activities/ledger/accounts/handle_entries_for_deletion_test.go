package accounts

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/logicorg/ivfs/finance/activities/ledger/internal/deletion"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// entriesForDeletionFixture inserts what the cases of HandleEntriesForDeletion
// build on. Kept entries stay behind, so every case inserts accounts of its
// own names and the tables are dropped after the test.
type entriesForDeletionFixture struct {
	s           *UnitTestSuite
	currency    *model.Currency
	institution *model.Institution
	asset       *model.Asset
	salary      *model.LedgerAccount
	groceries   *model.LedgerAccount
	openings    *model.LedgerAccount
	clearing    *model.LedgerAccount
	// bank is a balance that survives every case, and broker a financial
	// account that does.
	bank   *model.LedgerAccount
	broker *model.FinancialAccount
}

func newEntriesForDeletionFixture(s *UnitTestSuite) *entriesForDeletionFixture {
	t := s.T()
	f := &entriesForDeletionFixture{
		s:           s,
		currency:    testhelpers.Euro(),
		institution: &model.Institution{Name: "Revolut", Kind: "bank"},
		asset:       &model.Asset{Name: "AAPL", Kind: model.AssetKindStock},
	}
	temporaltesting.Insert(t, s.DB, f.currency)
	temporaltesting.Insert(t, s.DB, f.institution)
	temporaltesting.Insert(t, s.DB, f.asset)
	f.salary = f.ledgerAccount(t, "Salary", model.LedgerAccountIncome, nil)
	f.groceries = f.ledgerAccount(t, "Groceries", model.LedgerAccountExpense, nil)
	f.openings = f.ledgerAccount(t, lt.OpeningBalancesAccountName, model.LedgerAccountEquity, nil)
	f.clearing = f.ledgerAccount(t, lt.FXClearingAccountName, model.LedgerAccountAsset, nil)
	f.bank = f.ledgerAccount(t, "Bank cash", model.LedgerAccountAsset, nil)
	f.broker = f.financialAccount(t, "Broker")
	return f
}

func (f *entriesForDeletionFixture) ledgerAccount(t *testing.T, name string, kind model.LedgerAccountKind, financialAccountID *int64) *model.LedgerAccount {
	account := &model.LedgerAccount{Name: name, CurrencyID: f.currency.ID, Kind: kind, FinancialAccountID: financialAccountID}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

func (f *entriesForDeletionFixture) financialAccount(t *testing.T, name string) *model.FinancialAccount {
	account := &model.FinancialAccount{InstitutionID: f.institution.ID, Name: name}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

// entry records a posted entry of 100 from credit to debit, with the
// movements.
func (f *entriesForDeletionFixture) entry(t *testing.T, debit, credit *model.LedgerAccount, movements ...*model.PositionMovement) *model.JournalEntry {
	entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)}
	temporaltesting.InsertPostedJournalEntryWithMovements(t, f.s.DB, entry, []model.Posting{
		{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
	}, movements)
	return entry
}

// reversal records the entry that reverses entry, from debit to credit.
func (f *entriesForDeletionFixture) reversal(t *testing.T, entry *model.JournalEntry, debit, credit *model.LedgerAccount) *model.JournalEntry {
	reversal := &model.JournalEntry{OccurredAt: entry.OccurredAt, ReversalOfEntryID: &entry.ID}
	temporaltesting.InsertPostedJournalEntry(t, f.s.DB, reversal,
		model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
	)
	return reversal
}

// draft records a draft entry of 100 from credit to debit.
func (f *entriesForDeletionFixture) draft(t *testing.T, debit, credit *model.LedgerAccount) *model.JournalEntry {
	entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), Status: model.JournalEntryDraft}
	temporaltesting.Insert(t, f.s.DB, entry)
	postings := []model.Posting{
		{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		{JournalEntryID: entry.ID, LineNumber: 2, LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
	}
	temporaltesting.Insert(t, f.s.DB, &postings)
	return entry
}

// units is a movement of 5 units of the asset in account.
func (f *entriesForDeletionFixture) units(account *model.FinancialAccount) *model.PositionMovement {
	return &model.PositionMovement{FinancialAccountID: account.ID, AssetID: f.asset.ID, UnitsDelta: decimal.NewFromInt(5)}
}

func (f *entriesForDeletionFixture) strayLedgerAccount(t *testing.T, kind model.LedgerAccountKind) *model.LedgerAccount {
	var id int64
	require.NoError(t, f.s.DB.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) (err error) {
		id, err = deletion.StrayLedgerAccountTx(ctx, tx, kind, f.currency.ID)
		return err
	}))
	account := &model.LedgerAccount{}
	temporaltesting.Retrieve(t, f.s.DB, account, id)
	return account
}

func (f *entriesForDeletionFixture) strayFinancialAccount(t *testing.T) *model.FinancialAccount {
	var id int64
	require.NoError(t, f.s.DB.RunInTx(t.Context(), nil, func(ctx context.Context, tx bun.Tx) (err error) {
		id, err = deletion.StrayFinancialAccountTx(ctx, tx)
		return err
	}))
	account := &model.FinancialAccount{}
	temporaltesting.Retrieve(t, f.s.DB, account, id)
	return account
}

// entryOutcome is what a batch should leave of an entry: nothing, or postings
// on the named ledger accounts, in line order, and movements in the named
// financial accounts, in id order.
type entryOutcome struct {
	entry     *model.JournalEntry
	deleted   bool
	postings  []string
	movements []string
}

func (f *entriesForDeletionFixture) requireOutcome(t *testing.T, want entryOutcome) {
	t.Helper()

	var entry model.JournalEntry
	err := f.s.DB.NewSelect().Model(&entry).
		Relation("Postings", func(q *bun.SelectQuery) *bun.SelectQuery { return q.OrderExpr("p.line_number") }).
		Relation("Postings.LedgerAccount").
		Relation("PositionMovements", func(q *bun.SelectQuery) *bun.SelectQuery { return q.OrderExpr("pm.id") }).
		Relation("PositionMovements.FinancialAccount").
		Where("je.id = ?", want.entry.ID).
		Scan(t.Context())
	if want.deleted {
		require.ErrorIs(t, err, sql.ErrNoRows, "entry %d is deleted", want.entry.ID)
		return
	}
	require.NoError(t, err)

	var postings, movements []string
	for _, posting := range entry.Postings {
		postings = append(postings, posting.LedgerAccount.Name)
	}
	for _, movement := range entry.PositionMovements {
		movements = append(movements, movement.FinancialAccount.Name)
	}
	require.Equal(t, want.postings, postings, "the ledger accounts of entry %d", entry.ID)
	require.Equal(t, want.movements, movements, "the financial accounts of entry %d", entry.ID)
}

func (s *UnitTestSuite) TestHandleEntriesForDeletion() {
	f := newEntriesForDeletionFixture(s)
	missing := int64(999999)
	ledgerTarget := func(account *model.LedgerAccount) lt.DeletionTarget {
		return lt.DeletionTarget{LedgerAccountID: &account.ID}
	}
	financialTarget := func(account *model.FinancialAccount) lt.DeletionTarget {
		return lt.DeletionTarget{FinancialAccountID: &account.ID}
	}

	tests := map[string]struct {
		// setup inserts the case's accounts and entries, and returns what to empty
		// and what should be left of the entries.
		setup func(t *testing.T) (lt.DeletionTarget, []entryOutcome)
		// limit is 100 unless set.
		limit   int
		handled int
		errStr  string
	}{
		"an expense paid from the account is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				wallet := f.ledgerAccount(t, "Wallet", model.LedgerAccountAsset, nil)
				return ledgerTarget(wallet), []entryOutcome{{entry: f.entry(t, f.groceries, wallet), deleted: true}}
			},
			handled: 1,
		},
		"an opening balance is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				piggy := f.ledgerAccount(t, "Piggy bank", model.LedgerAccountAsset, nil)
				return ledgerTarget(piggy), []entryOutcome{{entry: f.entry(t, piggy, f.openings), deleted: true}}
			},
			handled: 1,
		},
		"a conversion through FX clearing is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				travel := f.ledgerAccount(t, "Travel cash", model.LedgerAccountAsset, nil)
				return ledgerTarget(travel), []entryOutcome{{entry: f.entry(t, travel, f.clearing), deleted: true}}
			},
			handled: 1,
		},
		"an entry against a stray is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				old := f.ledgerAccount(t, "Old cash", model.LedgerAccountAsset, nil)
				untracked := f.strayLedgerAccount(t, model.LedgerAccountAsset)
				return ledgerTarget(old), []entryOutcome{{entry: f.entry(t, old, untracked), deleted: true}}
			},
			handled: 1,
		},
		"a transfer from a surviving balance is kept, on untracked accounts": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				revolut := f.ledgerAccount(t, "Revolut cash", model.LedgerAccountAsset, nil)
				return ledgerTarget(revolut), []entryOutcome{
					{entry: f.entry(t, revolut, f.bank), postings: []string{lt.UntrackedAccountsName, "Bank cash"}},
				}
			},
			handled: 1,
		},
		"a category's expense from a surviving balance is kept, on uncategorized": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				coffee := f.ledgerAccount(t, "Coffee", model.LedgerAccountExpense, nil)
				return ledgerTarget(coffee), []entryOutcome{
					{entry: f.entry(t, coffee, f.bank), postings: []string{lt.UncategorizedAccountName, "Bank cash"}},
				}
			},
			handled: 1,
		},
		"a draft is deleted even when it touches a surviving balance": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				pending := f.ledgerAccount(t, "Pending cash", model.LedgerAccountAsset, nil)
				return ledgerTarget(pending), []entryOutcome{{entry: f.draft(t, pending, f.bank), deleted: true}}
			},
			handled: 1,
		},
		"a reversal is deleted with the entry it reverses": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				gym := f.ledgerAccount(t, "Gym cash", model.LedgerAccountAsset, nil)
				entry := f.entry(t, f.groceries, gym)
				return ledgerTarget(gym), []entryOutcome{
					{entry: entry, deleted: true},
					{entry: f.reversal(t, entry, f.groceries, gym), deleted: true},
				}
			},
			handled: 2,
		},
		"a reversal of a kept entry is kept": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				loan := f.ledgerAccount(t, "Loan cash", model.LedgerAccountAsset, nil)
				entry := f.entry(t, loan, f.bank)
				return ledgerTarget(loan), []entryOutcome{
					{entry: entry, postings: []string{lt.UntrackedAccountsName, "Bank cash"}},
					{entry: f.reversal(t, entry, loan, f.bank), postings: []string{"Bank cash", lt.UntrackedAccountsName}},
				}
			},
			handled: 2,
		},
		"an entry with units on a surviving financial account is kept, with its units": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				espp := f.ledgerAccount(t, "ESPP", model.LedgerAccountAsset, nil)
				return ledgerTarget(espp), []entryOutcome{
					{entry: f.entry(t, espp, f.salary, f.units(f.broker)), postings: []string{lt.UntrackedAccountsName, "Salary"}, movements: []string{"Broker"}},
				}
			},
			handled: 1,
		},
		"a batch handles at most its limit, in id order": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				card := f.ledgerAccount(t, "Card cash", model.LedgerAccountAsset, nil)
				return ledgerTarget(card), []entryOutcome{
					{entry: f.entry(t, f.groceries, card), deleted: true},
					{entry: f.entry(t, f.groceries, card), postings: []string{"Groceries", "Card cash"}},
				}
			},
			limit:   1,
			handled: 1,
		},
		"a ledger account without entries": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				return ledgerTarget(f.ledgerAccount(t, "Unused cash", model.LedgerAccountAsset, nil)), nil
			},
		},
		"a ledger account that does not exist": {
			setup: func(*testing.T) (lt.DeletionTarget, []entryOutcome) {
				return lt.DeletionTarget{LedgerAccountID: &missing}, nil
			},
		},
		"a shared ledger account with entries": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				return ledgerTarget(f.clearing), []entryOutcome{
					{entry: f.entry(t, f.bank, f.clearing), postings: []string{"Bank cash", lt.FXClearingAccountName}},
				}
			},
			errStr: `"FX clearing" is shared by the ledger and cannot be deleted while it has postings`,
		},
		"a transfer between two of a financial account's ledger accounts is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				account := f.financialAccount(t, "Revolut EUR")
				cash := f.ledgerAccount(t, "Revolut EUR cash", model.LedgerAccountAsset, &account.ID)
				savings := f.ledgerAccount(t, "Revolut EUR savings", model.LedgerAccountAsset, &account.ID)
				return financialTarget(account), []entryOutcome{{entry: f.entry(t, savings, cash), deleted: true}}
			},
			handled: 1,
		},
		"a trade inside a financial account is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				account := f.financialAccount(t, "IBKR")
				investments := f.ledgerAccount(t, "IBKR investments", model.LedgerAccountAsset, &account.ID)
				cash := f.ledgerAccount(t, "IBKR cash", model.LedgerAccountAsset, &account.ID)
				return financialTarget(account), []entryOutcome{{entry: f.entry(t, investments, cash, f.units(account)), deleted: true}}
			},
			handled: 1,
		},
		"an entry that reaches a financial account only through units is deleted": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				account := f.financialAccount(t, "Schwab")
				return financialTarget(account), []entryOutcome{{entry: f.entry(t, f.groceries, f.salary, f.units(account)), deleted: true}}
			},
			handled: 1,
		},
		"a trade paid from a surviving balance is kept, with its units on the stray": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				account := f.financialAccount(t, "Degiro")
				investments := f.ledgerAccount(t, "Degiro investments", model.LedgerAccountAsset, &account.ID)
				return financialTarget(account), []entryOutcome{{
					entry:     f.entry(t, investments, f.bank, f.units(account)),
					postings:  []string{lt.UntrackedAccountsName, "Bank cash"},
					movements: []string{lt.UntrackedAccountsName},
				}}
			},
			handled: 1,
		},
		"the stray financial account with entries": {
			setup: func(t *testing.T) (lt.DeletionTarget, []entryOutcome) {
				stray := f.strayFinancialAccount(t)
				return financialTarget(stray), []entryOutcome{{
					entry:     f.entry(t, f.groceries, f.salary, f.units(stray)),
					postings:  []string{"Groceries", "Salary"},
					movements: []string{lt.UntrackedAccountsName},
				}}
			},
			errStr: `"Untracked accounts" holds the lines of deleted accounts and cannot be deleted while it has postings or movements`,
		},
		"a financial account that does not exist": {
			setup: func(*testing.T) (lt.DeletionTarget, []entryOutcome) {
				return lt.DeletionTarget{FinancialAccountID: &missing}, nil
			},
		},
		"a limit that is not positive": {
			setup: func(*testing.T) (lt.DeletionTarget, []entryOutcome) {
				return lt.DeletionTarget{LedgerAccountID: &missing}, nil
			},
			limit:  -1,
			errStr: "a deletion batch limit must be positive",
		},
		"both a ledger and a financial account": {
			setup: func(*testing.T) (lt.DeletionTarget, []entryOutcome) {
				return lt.DeletionTarget{LedgerAccountID: &missing, FinancialAccountID: &missing}, nil
			},
			errStr: "a deletion target is either a ledger account or a financial account",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()
			target, outcomes := tc.setup(t)
			limit := tc.limit
			if limit == 0 {
				limit = 100
			}

			result, err := s.Env.ExecuteActivity(s.aa.HandleEntriesForDeletion, lt.DeletionBatchInput{Target: target, Limit: limit})

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
			} else {
				require.NoError(t, err)
				var got lt.DeletionBatchResult
				require.NoError(t, result.Get(&got))
				require.Equal(t, tc.handled, got.Handled)
			}
			for _, outcome := range outcomes {
				f.requireOutcome(t, outcome)
			}
		})
	}
}

// A batch run again after it handled everything handles nothing, and leaves
// the kept entries' lines where the first run moved them.
func (s *UnitTestSuite) TestHandleEntriesForDeletion_Idempotency() {
	t := s.T()
	f := newEntriesForDeletionFixture(s)
	account := f.financialAccount(t, "Revolut EUR")
	cash := f.ledgerAccount(t, "Revolut EUR cash", model.LedgerAccountAsset, &account.ID)
	fees := f.ledgerAccount(t, "Revolut EUR fees", model.LedgerAccountExpense, &account.ID)
	outcomes := []entryOutcome{
		{entry: f.entry(t, fees, cash), deleted: true},
		{entry: f.entry(t, cash, f.bank, f.units(account)), postings: []string{lt.UntrackedAccountsName, "Bank cash"}, movements: []string{lt.UntrackedAccountsName}},
		{entry: f.entry(t, fees, f.bank), postings: []string{lt.UncategorizedAccountName, "Bank cash"}},
	}
	input := lt.DeletionBatchInput{Target: lt.DeletionTarget{FinancialAccountID: &account.ID}, Limit: 100}

	for run, handled := range []int{3, 0} {
		result, err := s.Env.ExecuteActivity(s.aa.HandleEntriesForDeletion, input)
		require.NoError(t, err)
		var got lt.DeletionBatchResult
		require.NoError(t, result.Get(&got))
		require.Equal(t, handled, got.Handled, "run %d", run+1)
		for _, outcome := range outcomes {
			f.requireOutcome(t, outcome)
		}
	}
}
