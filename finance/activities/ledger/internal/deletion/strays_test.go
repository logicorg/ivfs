package deletion

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// strayFixtures are the currencies and the institution the cases build on.
type strayFixtures struct {
	euro, dollar *model.Currency
	bank         *model.FinancialAccount
}

func seedStrayFixtures(s *UnitTestSuite, tx bun.Tx) strayFixtures {
	f := strayFixtures{euro: testhelpers.Euro(), dollar: testhelpers.USDollar()}
	insertTx(s.T(), tx, f.euro)
	insertTx(s.T(), tx, f.dollar)
	institution := &model.Institution{Name: "Revolut", Kind: "bank"}
	insertTx(s.T(), tx, institution)
	f.bank = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Revolut Personal"}
	insertTx(s.T(), tx, f.bank)
	return f
}

// Test_StrayLedgerAccountTx checks which stray receives the postings of each
// kind, and that a stray is found by name, case-insensitively, kind, currency,
// and being top-level with no financial account, and created only when there
// is none.
func (s *UnitTestSuite) Test_StrayLedgerAccountTx() {
	closedAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		kind model.LedgerAccountKind
		// existing inserts the accounts there are before the call. With
		// wantExisting, the first of them is the stray to return; otherwise a
		// new one is.
		existing     func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount
		dollar       bool
		wantExisting bool
		wantName     string
		wantKind     model.LedgerAccountKind
	}{
		"an asset goes to untracked accounts": {
			kind: model.LedgerAccountAsset, wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
		"a liability goes to untracked accounts": {
			kind: model.LedgerAccountLiability, wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
		"an equity account goes to untracked accounts": {
			kind: model.LedgerAccountEquity, wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
		"an expense goes to uncategorized expense": {
			kind: model.LedgerAccountExpense, wantName: lt.UncategorizedAccountName, wantKind: model.LedgerAccountExpense,
		},
		"an income goes to uncategorized income": {
			kind: model.LedgerAccountIncome, wantName: lt.UncategorizedAccountName, wantKind: model.LedgerAccountIncome,
		},
		"an existing stray in another casing": {
			kind: model.LedgerAccountAsset,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				return inserted(insert, &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "untracked ACCOUNTS", Kind: model.LedgerAccountEquity})
			},
			wantExisting: true, wantName: "untracked ACCOUNTS", wantKind: model.LedgerAccountEquity,
		},
		"an existing closed stray": {
			kind: model.LedgerAccountExpense,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				return inserted(insert, &model.LedgerAccount{CurrencyID: f.euro.ID, Name: lt.UncategorizedAccountName, Kind: model.LedgerAccountExpense, ClosedAt: &closedAt})
			},
			wantExisting: true, wantName: lt.UncategorizedAccountName, wantKind: model.LedgerAccountExpense,
		},
		"a namesake in another currency": {
			kind: model.LedgerAccountAsset, dollar: true,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				return inserted(insert, &model.LedgerAccount{CurrencyID: f.euro.ID, Name: lt.UntrackedAccountsName, Kind: model.LedgerAccountEquity})
			},
			wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
		"a namesake of the other category kind": {
			kind: model.LedgerAccountIncome,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				return inserted(insert, &model.LedgerAccount{CurrencyID: f.euro.ID, Name: lt.UncategorizedAccountName, Kind: model.LedgerAccountExpense})
			},
			wantName: lt.UncategorizedAccountName, wantKind: model.LedgerAccountIncome,
		},
		"a namesake under a financial account": {
			kind: model.LedgerAccountAsset,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				return inserted(insert, &model.LedgerAccount{FinancialAccountID: &f.bank.ID, CurrencyID: f.euro.ID, Name: lt.UntrackedAccountsName, Kind: model.LedgerAccountEquity})
			},
			wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
		"a namesake under a parent": {
			kind: model.LedgerAccountAsset,
			existing: func(insert func(*model.LedgerAccount), f strayFixtures) []*model.LedgerAccount {
				parent := &model.LedgerAccount{CurrencyID: f.euro.ID, Name: "Equity", Kind: model.LedgerAccountEquity}
				insert(parent)
				return inserted(insert, &model.LedgerAccount{ParentID: &parent.ID, CurrencyID: f.euro.ID, Name: lt.UntrackedAccountsName, Kind: model.LedgerAccountEquity})
			},
			wantName: lt.UntrackedAccountsName, wantKind: model.LedgerAccountEquity,
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			s.inRolledBackTx(func(tx bun.Tx) {
				f := seedStrayFixtures(s, tx)
				var existing []*model.LedgerAccount
				if tc.existing != nil {
					existing = tc.existing(func(account *model.LedgerAccount) { insertTx(s.T(), tx, account) }, f)
				}
				currency := f.euro
				if tc.dollar {
					currency = f.dollar
				}

				id, err := StrayLedgerAccountTx(s.T().Context(), tx, tc.kind, currency.ID)
				require.NoError(s.T(), err)
				again, err := StrayLedgerAccountTx(s.T().Context(), tx, tc.kind, currency.ID)
				require.NoError(s.T(), err)
				assert.Equal(s.T(), id, again, "a second call returns the same stray")

				for n, account := range existing {
					if tc.wantExisting && n == 0 {
						assert.Equal(s.T(), account.ID, id)
					} else {
						assert.NotEqual(s.T(), account.ID, id)
					}
				}
				var stray model.LedgerAccount
				require.NoError(s.T(), tx.NewSelect().Model(&stray).Where("la.id = ?", id).Scan(s.T().Context()))
				assert.Equal(s.T(), tc.wantName, stray.Name)
				assert.Equal(s.T(), tc.wantKind, stray.Kind)
				assert.Equal(s.T(), currency.ID, stray.CurrencyID)
				assert.Nil(s.T(), stray.ParentID)
				assert.Nil(s.T(), stray.FinancialAccountID)
			})
		})
	}
}

// inserted inserts the accounts and returns them.
func inserted(insert func(*model.LedgerAccount), accounts ...*model.LedgerAccount) []*model.LedgerAccount {
	for _, account := range accounts {
		insert(account)
	}
	return accounts
}

// Test_StrayLedgerAccountTx_Concurrent checks that two transactions asking for
// a missing stray at once get the same one, and neither is aborted.
func (s *UnitTestSuite) Test_StrayLedgerAccountTx_Concurrent() {
	ctx := s.T().Context()
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)

	first, err := s.DB.BeginTx(ctx, nil)
	require.NoError(s.T(), err)
	defer func() { _ = first.Rollback() }()
	firstID, err := StrayLedgerAccountTx(ctx, first, model.LedgerAccountExpense, euro.ID)
	require.NoError(s.T(), err)

	// The second insert waits on the first transaction's row until it commits.
	secondID := make(chan int64, 1)
	go func() {
		defer close(secondID)
		second, err := s.DB.BeginTx(ctx, nil)
		if !assert.NoError(s.T(), err) {
			return
		}
		defer func() { _ = second.Rollback() }()
		id, err := StrayLedgerAccountTx(ctx, second, model.LedgerAccountExpense, euro.ID)
		if assert.NoError(s.T(), err) && assert.NoError(s.T(), second.Commit()) {
			secondID <- id
		}
	}()
	time.Sleep(100 * time.Millisecond)
	require.NoError(s.T(), first.Commit())

	assert.Equal(s.T(), firstID, <-secondID)
	temporaltesting.Delete(s.T(), s.DB, (*model.LedgerAccount)(nil), firstID)
	temporaltesting.Delete(s.T(), s.DB, (*model.Currency)(nil), euro.ID)
}

// Test_StrayLedgerAccountsTx checks that each account gets the stray of its
// kind and currency, and that accounts with the same stray share it.
func (s *UnitTestSuite) Test_StrayLedgerAccountsTx() {
	s.inRolledBackTx(func(tx bun.Tx) {
		f := seedStrayFixtures(s, tx)
		account := func(name string, kind model.LedgerAccountKind, currency *model.Currency) *model.LedgerAccount {
			a := &model.LedgerAccount{CurrencyID: currency.ID, Name: name, Kind: kind}
			insertTx(s.T(), tx, a)
			return a
		}
		groceries := account("Groceries", model.LedgerAccountExpense, f.euro)
		cash := account("Cash", model.LedgerAccountAsset, f.euro)
		card := account("Card balance", model.LedgerAccountLiability, f.euro)
		salary := account("Salary", model.LedgerAccountIncome, f.euro)
		dollars := account("Dollar cash", model.LedgerAccountAsset, f.dollar)

		strays, err := StrayLedgerAccountsTx(s.T().Context(), tx, []*model.LedgerAccount{groceries, cash, card, salary, dollars})
		require.NoError(s.T(), err)

		want := map[*model.LedgerAccount]struct {
			name     string
			kind     model.LedgerAccountKind
			currency *model.Currency
		}{
			groceries: {lt.UncategorizedAccountName, model.LedgerAccountExpense, f.euro},
			cash:      {lt.UntrackedAccountsName, model.LedgerAccountEquity, f.euro},
			card:      {lt.UntrackedAccountsName, model.LedgerAccountEquity, f.euro},
			salary:    {lt.UncategorizedAccountName, model.LedgerAccountIncome, f.euro},
			dollars:   {lt.UntrackedAccountsName, model.LedgerAccountEquity, f.dollar},
		}
		require.Len(s.T(), strays, len(want))
		for account, w := range want {
			var stray model.LedgerAccount
			require.NoError(s.T(), tx.NewSelect().Model(&stray).Where("la.id = ?", strays[account.ID]).Scan(s.T().Context()))
			assert.Equal(s.T(), w.name, stray.Name, account.Name)
			assert.Equal(s.T(), w.kind, stray.Kind, account.Name)
			assert.Equal(s.T(), w.currency.ID, stray.CurrencyID, account.Name)
		}
		assert.Equal(s.T(), strays[cash.ID], strays[card.ID], "a balance and a liability in one currency share the stray")
	})
}

// Test_StrayFinancialAccountTx checks that the stray financial account is
// found by its name and its institution's, case-insensitively, and that it and
// its institution are created only when missing.
func (s *UnitTestSuite) Test_StrayFinancialAccountTx() {
	tests := map[string]struct {
		// existing inserts what there is before the call, and returns the stray
		// to expect, or nil for a new one.
		existing func(tx bun.Tx) *model.FinancialAccount
	}{
		"nothing yet": {},
		"the institution alone, in another casing": {
			existing: func(tx bun.Tx) *model.FinancialAccount {
				insertTx(s.T(), tx, &model.Institution{Name: "UNTRACKED", Kind: "bank"})
				return nil
			},
		},
		"both, in another casing": {
			existing: func(tx bun.Tx) *model.FinancialAccount {
				institution := &model.Institution{Name: "untracked", Kind: "untracked"}
				insertTx(s.T(), tx, institution)
				stray := &model.FinancialAccount{InstitutionID: institution.ID, Name: "untracked accounts"}
				insertTx(s.T(), tx, stray)
				return stray
			},
		},
		"a namesake at another institution": {
			existing: func(tx bun.Tx) *model.FinancialAccount {
				institution := &model.Institution{Name: "Revolut", Kind: "bank"}
				insertTx(s.T(), tx, institution)
				insertTx(s.T(), tx, &model.FinancialAccount{InstitutionID: institution.ID, Name: lt.UntrackedAccountsName})
				return nil
			},
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			s.inRolledBackTx(func(tx bun.Tx) {
				var want *model.FinancialAccount
				if tc.existing != nil {
					want = tc.existing(tx)
				}

				id, err := StrayFinancialAccountTx(s.T().Context(), tx)
				require.NoError(s.T(), err)
				again, err := StrayFinancialAccountTx(s.T().Context(), tx)
				require.NoError(s.T(), err)
				assert.Equal(s.T(), id, again, "a second call returns the same stray")

				if want != nil {
					assert.Equal(s.T(), want.ID, id)
				}
				var stray model.FinancialAccount
				require.NoError(s.T(), tx.NewSelect().Model(&stray).Where("fa.id = ?", id).Scan(s.T().Context()))
				var institution model.Institution
				require.NoError(s.T(), tx.NewSelect().Model(&institution).Where("i.id = ?", stray.InstitutionID).Scan(s.T().Context()))
				assert.True(s.T(), lt.IsUntrackedFinancialAccountName(stray.Name), stray.Name)
				assert.True(s.T(), lt.IsUntrackedInstitution(institution.Name), institution.Name)
				count, err := tx.NewSelect().Model((*model.Institution)(nil)).Where("lower(i.name) = lower(?)", lt.UntrackedInstitutionName).Count(s.T().Context())
				require.NoError(s.T(), err)
				assert.Equal(s.T(), 1, count, "one untracked institution")
			})
		})
	}
}
