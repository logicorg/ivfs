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

// closeForDeletionFixture inserts what the cases of CloseAccountForDeletion
// build on. Posted entries cannot be deleted, so every case inserts accounts
// of its own names and the tables are dropped after the test.
type closeForDeletionFixture struct {
	s           *UnitTestSuite
	currency    *model.Currency
	institution *model.Institution
	asset       *model.Asset
	// salary is the other side of every entry.
	salary *model.LedgerAccount
}

func newCloseForDeletionFixture(s *UnitTestSuite) *closeForDeletionFixture {
	t := s.T()
	f := &closeForDeletionFixture{
		s:           s,
		currency:    testhelpers.Euro(),
		institution: &model.Institution{Name: "Revolut", Kind: "bank"},
		asset:       &model.Asset{Name: "AAPL", Kind: model.AssetKindStock},
	}
	temporaltesting.Insert(t, s.DB, f.currency)
	temporaltesting.Insert(t, s.DB, f.institution)
	temporaltesting.Insert(t, s.DB, f.asset)
	f.salary = f.ledgerAccount(t, "Salary", model.LedgerAccountIncome, nil)
	return f
}

func (f *closeForDeletionFixture) ledgerAccount(t *testing.T, name string, kind model.LedgerAccountKind, financialAccountID *int64) *model.LedgerAccount {
	account := &model.LedgerAccount{Name: name, CurrencyID: f.currency.ID, Kind: kind, FinancialAccountID: financialAccountID}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

func (f *closeForDeletionFixture) financialAccount(t *testing.T, institutionID int64, name string) *model.FinancialAccount {
	account := &model.FinancialAccount{InstitutionID: institutionID, Name: name}
	temporaltesting.Insert(t, f.s.DB, account)
	return account
}

// post records an entry of 100 from salary to account.
func (f *closeForDeletionFixture) post(t *testing.T, account *model.LedgerAccount) {
	temporaltesting.InsertPostedJournalEntry(t, f.s.DB, &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
		model.Posting{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: f.salary.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
	)
}

// hold records 5 units of the asset on account.
func (f *closeForDeletionFixture) hold(t *testing.T, account *model.FinancialAccount) {
	temporaltesting.InsertPositionMovement(t, f.s.DB, &model.PositionMovement{
		FinancialAccountID: account.ID, AssetID: f.asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: time.Now().UTC().Add(-time.Hour),
	})
}

// closedAt returns the ClosedAt of the target's rows: the ledger account, or
// the financial account followed by its ledger accounts.
func (f *closeForDeletionFixture) closedAt(t *testing.T, target lt.DeletionTarget) []*time.Time {
	if target.LedgerAccountID != nil {
		var account model.LedgerAccount
		require.NoError(t, f.s.DB.NewSelect().Model(&account).Where("la.id = ?", *target.LedgerAccountID).Scan(t.Context()))
		return []*time.Time{account.ClosedAt}
	}
	var account model.FinancialAccount
	require.NoError(t, f.s.DB.NewSelect().Model(&account).Where("fa.id = ?", *target.FinancialAccountID).Scan(t.Context()))
	var ledgerAccounts []model.LedgerAccount
	require.NoError(t, f.s.DB.NewSelect().Model(&ledgerAccounts).Where("la.financial_account_id = ?", account.ID).OrderExpr("la.id").Scan(t.Context()))
	closed := []*time.Time{account.ClosedAt}
	for _, ledgerAccount := range ledgerAccounts {
		closed = append(closed, ledgerAccount.ClosedAt)
	}
	return closed
}

func (s *UnitTestSuite) TestCloseAccountForDeletion() {
	f := newCloseForDeletionFixture(s)
	missing := int64(999999)

	tests := map[string]struct {
		// target inserts the case's accounts and returns what to close.
		target func(t *testing.T) lt.DeletionTarget
		gone   bool
		errStr string
	}{
		"a ledger account with a balance": {
			target: func(t *testing.T) lt.DeletionTarget {
				cash := f.ledgerAccount(t, "Cash", model.LedgerAccountAsset, nil)
				f.post(t, cash)
				return lt.DeletionTarget{LedgerAccountID: &cash.ID}
			},
		},
		"a ledger account with child accounts": {
			target: func(t *testing.T) lt.DeletionTarget {
				parent := f.ledgerAccount(t, "Savings", model.LedgerAccountAsset, nil)
				child := &model.LedgerAccount{Name: "Savings pot", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset, ParentID: &parent.ID}
				temporaltesting.Insert(t, s.DB, child)
				return lt.DeletionTarget{LedgerAccountID: &parent.ID}
			},
			errStr: "has child accounts; delete them first",
		},
		"a shared ledger account with postings": {
			target: func(t *testing.T) lt.DeletionTarget {
				clearing := f.ledgerAccount(t, lt.FXClearingAccountName, model.LedgerAccountAsset, nil)
				f.post(t, clearing)
				return lt.DeletionTarget{LedgerAccountID: &clearing.ID}
			},
			errStr: `"FX clearing" is shared by the ledger and cannot be deleted while it has postings`,
		},
		"a shared ledger account without postings": {
			target: func(t *testing.T) lt.DeletionTarget {
				openings := f.ledgerAccount(t, lt.OpeningBalancesAccountName, model.LedgerAccountEquity, nil)
				return lt.DeletionTarget{LedgerAccountID: &openings.ID}
			},
		},
		"a ledger account that does not exist": {
			target: func(*testing.T) lt.DeletionTarget { return lt.DeletionTarget{LedgerAccountID: &missing} },
			gone:   true,
		},
		"a financial account with units and a balance": {
			target: func(t *testing.T) lt.DeletionTarget {
				account := f.financialAccount(t, f.institution.ID, "Revolut EUR")
				cash := f.ledgerAccount(t, "Revolut cash", model.LedgerAccountAsset, &account.ID)
				f.ledgerAccount(t, "Revolut fees", model.LedgerAccountExpense, &account.ID)
				f.post(t, cash)
				f.hold(t, account)
				return lt.DeletionTarget{FinancialAccountID: &account.ID}
			},
		},
		"a financial account whose ledger account has a child outside it": {
			target: func(t *testing.T) lt.DeletionTarget {
				account := f.financialAccount(t, f.institution.ID, "Revolut USD")
				cash := f.ledgerAccount(t, "Revolut USD cash", model.LedgerAccountAsset, &account.ID)
				child := &model.LedgerAccount{Name: "Revolut USD pot", CurrencyID: f.currency.ID, Kind: model.LedgerAccountAsset, ParentID: &cash.ID}
				temporaltesting.Insert(t, s.DB, child)
				return lt.DeletionTarget{FinancialAccountID: &account.ID}
			},
			errStr: "has child accounts outside it; delete or move them first",
		},
		"the stray financial account with movements": {
			target: func(t *testing.T) lt.DeletionTarget {
				untracked := &model.Institution{Name: lt.UntrackedInstitutionName, Kind: "untracked"}
				temporaltesting.Insert(t, s.DB, untracked)
				stray := f.financialAccount(t, untracked.ID, lt.UntrackedAccountsName)
				f.hold(t, stray)
				return lt.DeletionTarget{FinancialAccountID: &stray.ID}
			},
			errStr: `"Untracked accounts" holds the lines of deleted accounts and cannot be deleted while it has postings or movements`,
		},
		"an account of the stray's name at another institution": {
			target: func(t *testing.T) lt.DeletionTarget {
				account := f.financialAccount(t, f.institution.ID, lt.UntrackedAccountsName)
				f.hold(t, account)
				return lt.DeletionTarget{FinancialAccountID: &account.ID}
			},
		},
		"a financial account that does not exist": {
			target: func(*testing.T) lt.DeletionTarget { return lt.DeletionTarget{FinancialAccountID: &missing} },
			gone:   true,
		},
		"both a ledger and a financial account": {
			target: func(*testing.T) lt.DeletionTarget {
				return lt.DeletionTarget{LedgerAccountID: &missing, FinancialAccountID: &missing}
			},
			errStr: "a deletion target is either a ledger account or a financial account",
		},
		"neither a ledger nor a financial account": {
			target: func(*testing.T) lt.DeletionTarget { return lt.DeletionTarget{} },
			errStr: "a deletion target is either a ledger account or a financial account",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			t := s.T()
			target := tc.target(t)

			result, err := s.Env.ExecuteActivity(s.aa.CloseAccountForDeletion, target)

			if tc.errStr != "" {
				require.ErrorContains(t, err, tc.errStr)
				if (target.LedgerAccountID == nil) != (target.FinancialAccountID == nil) {
					for _, closedAt := range f.closedAt(t, target) {
						require.Nil(t, closedAt, "a refused target stays open")
					}
				}
				return
			}
			require.NoError(t, err)
			var got lt.CloseAccountForDeletionResult
			require.NoError(t, result.Get(&got))
			require.Equal(t, tc.gone, got.Gone)
			if tc.gone {
				return
			}
			for _, closedAt := range f.closedAt(t, target) {
				require.NotNil(t, closedAt, "every row of the target is closed")
			}
		})
	}
}

// An account closed earlier, whether by a person or by a deletion that
// stopped, keeps its ClosedAt when the deletion runs again.
func (s *UnitTestSuite) TestCloseAccountForDeletion_Idempotency() {
	t := s.T()
	f := newCloseForDeletionFixture(s)
	account := f.financialAccount(t, f.institution.ID, "Revolut EUR")
	cash := f.ledgerAccount(t, "Revolut cash", model.LedgerAccountAsset, &account.ID)
	f.post(t, cash)
	earlier := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	fees := &model.LedgerAccount{Name: "Revolut fees", CurrencyID: f.currency.ID, Kind: model.LedgerAccountExpense, FinancialAccountID: &account.ID, ClosedAt: &earlier}
	temporaltesting.Insert(t, s.DB, fees)
	target := lt.DeletionTarget{FinancialAccountID: &account.ID}

	_, err := s.Env.ExecuteActivity(s.aa.CloseAccountForDeletion, target)
	require.NoError(t, err)
	first := f.closedAt(t, target)

	result, err := s.Env.ExecuteActivity(s.aa.CloseAccountForDeletion, target)
	require.NoError(t, err)
	var got lt.CloseAccountForDeletionResult
	require.NoError(t, result.Get(&got))
	require.False(t, got.Gone)

	second := f.closedAt(t, target)
	require.Len(t, second, 3)
	for i := range first {
		require.NotNil(t, second[i])
		require.True(t, first[i].Equal(*second[i]), "row %d keeps its ClosedAt", i)
	}
	require.True(t, earlier.Equal(*second[2]), "the fees account keeps the ClosedAt it had before the deletion")
}
