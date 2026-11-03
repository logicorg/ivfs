package accounts

import (
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// deleteFinancialAccountFixture inserts what the cases of DeleteFinancialAccount
// build on. Posted entries cannot be deleted, so every case inserts accounts of
// its own names and the tables are dropped after the test.
type deleteFinancialAccountFixture struct {
	s           *UnitTestSuite
	currency    *model.Currency
	institution *model.Institution
	asset       *model.Asset
	// bank is the other side of every entry.
	bank *model.LedgerAccount
}

func newDeleteFinancialAccountFixture(s *UnitTestSuite) *deleteFinancialAccountFixture {
	t := s.T()
	f := &deleteFinancialAccountFixture{
		s:           s,
		currency:    testhelpers.Euro(),
		institution: &model.Institution{Name: "Revolut", Kind: "bank"},
		asset:       &model.Asset{Name: "AAPL", Kind: model.AssetKindStock},
	}
	temporaltesting.Insert(t, s.DB, f.currency)
	temporaltesting.Insert(t, s.DB, f.institution)
	temporaltesting.Insert(t, s.DB, f.asset)
	f.bank = f.ledgerAccount(t, "Bank", nil, nil)
	return f
}

func (f *deleteFinancialAccountFixture) financialAccount(t *testing.T, name string) *model.FinancialAccount {
	account := &model.FinancialAccount{InstitutionID: f.institution.ID, Name: name}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

// ledgerAccount inserts an asset account named name, in financialAccount and
// under parent when they are not nil.
func (f *deleteFinancialAccountFixture) ledgerAccount(t *testing.T, name string, financialAccount *model.FinancialAccount, parent *model.LedgerAccount) *model.LedgerAccount {
	account := &model.LedgerAccount{Name: name, CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset}
	if financialAccount != nil {
		account.FinancialAccountID = &financialAccount.ID
	}
	if parent != nil {
		account.ParentID = &parent.ID
	}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

// post records an entry of 100 from bank to account.
func (f *deleteFinancialAccountFixture) post(t *testing.T, account *model.LedgerAccount) {
	temporaltesting.InsertPostedJournalEntry(t, f.s.DB, &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
		model.Posting{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: f.bank.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
	)
}

// hold records 5 units of the asset on account.
func (f *deleteFinancialAccountFixture) hold(t *testing.T, account *model.FinancialAccount) {
	temporaltesting.InsertPositionMovement(t, f.s.DB, &model.PositionMovement{
		FinancialAccountID: account.ID, AssetID: f.asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	})
}

// rows counts the financial account and its ledger accounts.
func (f *deleteFinancialAccountFixture) rows(t *testing.T, id int64) int {
	accounts, err := f.s.DB.NewSelect().Model((*model.FinancialAccount)(nil)).Where("fa.id = ?", id).Count(t.Context())
	require.NoError(t, err)
	ledgerAccounts, err := f.s.DB.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.financial_account_id = ?", id).Count(t.Context())
	require.NoError(t, err)
	return accounts + ledgerAccounts
}

// rowsOutside counts the ledger accounts outside every financial account.
func (f *deleteFinancialAccountFixture) rowsOutside(t *testing.T) int {
	count, err := f.s.DB.NewSelect().Model((*model.LedgerAccount)(nil)).Where("la.financial_account_id IS NULL").Count(t.Context())
	require.NoError(t, err)
	return count
}

func (s *UnitTestSuite) TestDeleteFinancialAccount() {
	f := newDeleteFinancialAccountFixture(s)

	tests := map[string]struct {
		// account inserts the case's accounts and returns the id to delete.
		account func(t *testing.T) int64
		errStr  string
	}{
		"an account without ledger accounts": {
			account: func(t *testing.T) int64 { return f.financialAccount(t, "Revolut bare").ID },
		},
		"an account with nested ledger accounts": {
			account: func(t *testing.T) int64 {
				account := f.financialAccount(t, "Revolut nested")
				cash := f.ledgerAccount(t, "Nested cash", account, nil)
				pot := f.ledgerAccount(t, "Nested pot", account, cash)
				f.ledgerAccount(t, "Nested pot savings", account, pot)
				f.ledgerAccount(t, "Nested pot holiday", account, pot)
				f.ledgerAccount(t, "Nested card", account, nil)
				return account.ID
			},
		},
		"an account that does not exist": {
			account: func(*testing.T) int64 { return 999999 },
		},
		"a ledger account with postings": {
			account: func(t *testing.T) int64 {
				account := f.financialAccount(t, "Revolut posted")
				cash := f.ledgerAccount(t, "Posted cash", account, nil)
				f.ledgerAccount(t, "Posted pot", account, cash)
				f.post(t, cash)
				return account.ID
			},
			errStr: "resource LedgerAccount conflict: still referenced by ledger.posting.ledger_account_id",
		},
		"an account with movements": {
			account: func(t *testing.T) int64 {
				account := f.financialAccount(t, "Revolut held")
				f.ledgerAccount(t, "Held cash", account, nil)
				f.hold(t, account)
				return account.ID
			},
			errStr: "resource FinancialAccount conflict: still referenced by asset.position_movement.financial_account_id",
		},
		"a ledger account with a child outside it": {
			account: func(t *testing.T) int64 {
				account := f.financialAccount(t, "Revolut parent")
				cash := f.ledgerAccount(t, "Parent cash", account, nil)
				f.ledgerAccount(t, "Parent leaf", account, nil)
				f.ledgerAccount(t, "Parent cash pot", nil, cash)
				return account.ID
			},
			errStr: "has child accounts outside it; delete or move them first",
		},
		"an id that is not positive": {
			account: func(*testing.T) int64 { return 0 },
			errStr:  "validation error",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()
			id := tc.account(t)
			before := f.rows(t, id)

			_, err := s.Env.ExecuteActivity(s.aa.DeleteFinancialAccount, lt.DeleteFinancialAccountInput{FinancialAccountID: id})

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
				require.Equal(t, before, f.rows(t, id), "a refused deletion deletes nothing")
				return
			}
			require.NoError(t, err)
			require.Zero(t, f.rows(t, id))
		})
	}
}

// A retry after the rows are gone finds the account already deleted.
func (s *UnitTestSuite) TestDeleteFinancialAccount_Idempotency() {
	t := s.T()
	f := newDeleteFinancialAccountFixture(s)
	account := f.financialAccount(t, "Revolut EUR")
	cash := f.ledgerAccount(t, "Revolut cash", account, nil)
	f.ledgerAccount(t, "Revolut pot", account, cash)
	input := lt.DeleteFinancialAccountInput{FinancialAccountID: account.ID}

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.aa.DeleteFinancialAccount, input)
		require.NoError(t, err)
		require.Zero(t, f.rows(t, account.ID))
	}
	require.Equal(t, 1, f.rowsOutside(t), "the other ledger accounts stay")
}
