package accounts

import (
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestRetrieveAndListLedgerAccounts() {
	currency := &model.Currency{Code: "GBP", Name: "Pound Sterling", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	financialAccount := &model.FinancialAccount{InstitutionID: 1, Name: "Bank"}
	institution := &model.Institution{Name: "Bank", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	financialAccount.InstitutionID = institution.ID
	temporaltesting.Insert(s.T(), s.DB, financialAccount)
	shared := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, shared)
	active := &model.LedgerAccount{Name: "Cash", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset, FinancialAccountID: &financialAccount.ID}
	temporaltesting.Insert(s.T(), s.DB, active)
	closedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	closed := &model.LedgerAccount{Name: "Old", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
	temporaltesting.Insert(s.T(), s.DB, closed)

	activityResult, err := s.Env.ExecuteActivity(s.aa.RetrieveLedgerAccount, act.RetrieveLedgerAccountParams{ID: active.ID})
	require.NoError(s.T(), err)
	var retrieved act.RetrieveLedgerAccountResult
	require.NoError(s.T(), activityResult.Get(&retrieved))
	require.NotNil(s.T(), retrieved.LedgerAccount)
	assert.Equal(s.T(), active.ID, retrieved.LedgerAccount.ID)
	assert.Equal(s.T(), currency.Code, retrieved.LedgerAccount.Currency.Code)

	activityResult, err = s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, act.ListLedgerAccountsParams{FinancialAccountID: &financialAccount.ID})
	require.NoError(s.T(), err)
	var listed act.ListLedgerAccountsResult
	require.NoError(s.T(), activityResult.Get(&listed))
	require.Len(s.T(), listed.LedgerAccounts, 1)
	assert.Equal(s.T(), active.ID, listed.LedgerAccounts[0].ID)

	activityResult, err = s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, act.ListLedgerAccountsParams{IncludeClosed: true})
	require.NoError(s.T(), err)
	listed = act.ListLedgerAccountsResult{}
	require.NoError(s.T(), activityResult.Get(&listed))
	assert.Len(s.T(), listed.LedgerAccounts, 3)

	_, err = s.Env.ExecuteActivity(s.aa.RetrieveLedgerAccount, act.RetrieveLedgerAccountParams{ID: 9999})
	assert.Contains(s.T(), err.Error(), "resource LedgerAccount not found")
}

// TestListLedgerAccountsNameFilter covers NameContains, a case-insensitive
// substring match on the ledger account's own name, the way a caller
// resolving something like "expenses" to an account would.
func (s *UnitTestSuite) TestListLedgerAccountsNameFilter() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)

	expenses := &model.LedgerAccount{Name: lt.ExpensesAccountName, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, expenses)
	income := &model.LedgerAccount{Name: lt.IncomeAccountName, CurrencyID: currency.ID, Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, income)

	tests := map[string]struct {
		nameContains string
		expIDs       []int64
	}{
		"case-insensitive substring match": {
			nameContains: "expen",
			expIDs:       []int64{expenses.ID},
		},
		"no match returns empty": {
			nameContains: "nonexistent",
			expIDs:       []int64{},
		},
		"blank filter matches everything": {
			nameContains: "",
			expIDs:       []int64{expenses.ID, income.ID},
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, act.ListLedgerAccountsParams{NameContains: tc.nameContains})
			require.NoError(s.T(), err)

			var listed act.ListLedgerAccountsResult
			require.NoError(s.T(), activityResult.Get(&listed))
			gotIDs := make([]int64, 0, len(listed.LedgerAccounts))
			for _, a := range listed.LedgerAccounts {
				gotIDs = append(gotIDs, a.ID)
			}
			assert.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}

// TestListLedgerAccountsKindFilter covers Kinds, which keeps only accounts of
// the given kinds, such as the income and expense accounts without the asset
// accounts behind financial accounts.
func (s *UnitTestSuite) TestListLedgerAccountsKindFilter() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)

	cash := &model.LedgerAccount{Name: "Cash", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	expenses := &model.LedgerAccount{Name: lt.ExpensesAccountName, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	income := &model.LedgerAccount{Name: lt.IncomeAccountName, CurrencyID: currency.ID, Kind: model.LedgerAccountIncome}
	for _, account := range []*model.LedgerAccount{cash, expenses, income} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	tests := map[string]struct {
		kinds     []model.LedgerAccountKind
		expIDs    []int64
		wantError string
	}{
		"income and expense": {
			kinds:  []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense},
			expIDs: []int64{expenses.ID, income.ID},
		},
		"a single kind": {
			kinds:  []model.LedgerAccountKind{model.LedgerAccountAsset},
			expIDs: []int64{cash.ID},
		},
		"a kind with no accounts": {
			kinds:  []model.LedgerAccountKind{model.LedgerAccountLiability},
			expIDs: []int64{},
		},
		"no kinds matches everything": {
			expIDs: []int64{cash.ID, expenses.ID, income.ID},
		},
		"an unknown kind": {
			kinds:     []model.LedgerAccountKind{"clearly-invalid-kind"},
			wantError: "validation error",
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, act.ListLedgerAccountsParams{Kinds: tc.kinds})
			if tc.wantError != "" {
				require.ErrorContains(s.T(), err, tc.wantError)
				return
			}
			require.NoError(s.T(), err)

			var listed act.ListLedgerAccountsResult
			require.NoError(s.T(), activityResult.Get(&listed))
			gotIDs := make([]int64, 0, len(listed.LedgerAccounts))
			for _, a := range listed.LedgerAccounts {
				gotIDs = append(gotIDs, a.ID)
			}
			assert.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}

func (s *UnitTestSuite) TestRenameLedgerAccount() {
	currency := &model.Currency{Code: "CAD", Name: "Canadian Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	account := &model.LedgerAccount{Name: "Wallet", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, account)

	activityResult, err := s.Env.ExecuteActivity(s.aa.RenameLedgerAccount, act.RenameLedgerAccountInput{LedgerAccountID: account.ID, Name: " Purse "})
	require.NoError(s.T(), err)
	var mutation act.LedgerAccountMutationResult
	require.NoError(s.T(), activityResult.Get(&mutation))
	assert.Equal(s.T(), account.ID, mutation.LedgerAccountID)

	var stored model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.Equal(s.T(), "Purse", stored.Name)
}

func (s *UnitTestSuite) TestCloseLedgerAccountIsIdempotent() {
	currency := testhelpers.JapaneseYen()
	temporaltesting.Insert(s.T(), s.DB, currency)
	account := &model.LedgerAccount{Name: "Wallet", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, account)

	for range 2 {
		_, err := s.Env.ExecuteActivity(s.aa.CloseLedgerAccount, act.CloseLedgerAccountInput{LedgerAccountID: account.ID, ClosedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)})
		require.NoError(s.T(), err)
	}

	var stored model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.NotNil(s.T(), stored.ClosedAt)
	assert.Equal(s.T(), 2026, stored.ClosedAt.Year())
}

func (s *UnitTestSuite) TestDeleteLedgerAccountOnlyAllowsUnused() {
	currency := &model.Currency{Code: "AUD", Name: "Australian Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	used := &model.LedgerAccount{Name: "Wallet", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	unused := &model.LedgerAccount{Name: "Purse", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, used)
	temporaltesting.Insert(s.T(), s.DB, unused)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: used.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit})

	_, err := s.Env.ExecuteActivity(s.aa.DeleteLedgerAccount, act.DeleteLedgerAccountInput{LedgerAccountID: used.ID})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "has postings; close it instead")

	_, err = s.Env.ExecuteActivity(s.aa.DeleteLedgerAccount, act.DeleteLedgerAccountInput{LedgerAccountID: unused.ID})
	require.NoError(s.T(), err)

	assert.Zero(s.T(), s.CountWhere[model.LedgerAccount]("id = ?", unused.ID))
	assert.Equal(s.T(), 1, s.CountWhere[model.LedgerAccount]("id = ?", used.ID))
}

func int64Ptr(value int64) *int64 { return &value }
