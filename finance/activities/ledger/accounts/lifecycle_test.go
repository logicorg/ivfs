package accounts

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestCreateLedgerAccountHierarchyRules() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	otherCurrency := &model.Currency{Code: "USD", Name: "US Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, otherCurrency)

	root := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, root)

	result, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		ParentID: &root.ID, CurrencyID: currency.ID, Name: "Food", Kind: model.LedgerAccountExpense,
	})
	require.NoError(s.T(), err)
	var created lt.CreateLedgerAccountResult
	require.NoError(s.T(), result.Get(&created))

	var child model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&child).Where("id = ?", created.LedgerAccountID).Scan(s.T().Context()))
	assert.Equal(s.T(), root.ID, *child.ParentID)

	tests := map[string]struct {
		input   lt.CreateLedgerAccountInput
		wantErr string
	}{
		"parent kind differs": {
			input:   lt.CreateLedgerAccountInput{ParentID: &root.ID, CurrencyID: currency.ID, Name: "Salary", Kind: model.LedgerAccountIncome},
			wantErr: "same kind",
		},
		"parent currency differs": {
			input:   lt.CreateLedgerAccountInput{ParentID: &root.ID, CurrencyID: otherCurrency.ID, Name: "USD food", Kind: model.LedgerAccountExpense},
			wantErr: "same currency",
		},
		"missing parent": {
			input:   lt.CreateLedgerAccountInput{ParentID: int64Ptr(9999), CurrencyID: currency.ID, Name: "Missing", Kind: model.LedgerAccountExpense},
			wantErr: "resource LedgerAccount not found",
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, tc.input)
			require.Error(s.T(), err)
			assert.Contains(s.T(), err.Error(), tc.wantErr)
		})
	}
}

func (s *UnitTestSuite) TestCreateLedgerAccountRejectsDepthAndPostingParent() {
	currency := &model.Currency{Code: "USD", Name: "US Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)

	root := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, root)
	child := &model.LedgerAccount{Name: "Food", ParentID: &root.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, child)
	grandchild := &model.LedgerAccount{Name: "Groceries", ParentID: &child.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, grandchild)

	_, err := s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		ParentID: &grandchild.ID, CurrencyID: currency.ID, Name: "Produce", Kind: model.LedgerAccountExpense,
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot exceed depth 2")

	postingParent := &model.LedgerAccount{Name: "Posted", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, postingParent)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)
	posting := &model.Posting{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: postingParent.ID, Amount: decimal.NewFromInt(1), Side: model.PostingDebit}
	temporaltesting.Insert(s.T(), s.DB, posting)

	_, err = s.Env.ExecuteActivity(s.aa.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		ParentID: &postingParent.ID, CurrencyID: currency.ID, Name: "Child", Kind: model.LedgerAccountExpense,
	})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot become a grouping account")
}

func (s *UnitTestSuite) TestRetrieveAndListLedgerAccounts() {
	currency := &model.Currency{Code: "GBP", Name: "Pound Sterling", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	financialAccount := &model.FinancialAccount{InstitutionID: 1, Name: "Bank"}
	institution := &model.Institution{Name: "Bank", Kind: "bank"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	financialAccount.InstitutionID = institution.ID
	temporaltesting.Insert(s.T(), s.DB, financialAccount)
	root := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, root)
	active := &model.LedgerAccount{Name: "Food", ParentID: &root.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense, FinancialAccountID: &financialAccount.ID}
	temporaltesting.Insert(s.T(), s.DB, active)
	closedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	closed := &model.LedgerAccount{Name: "Old", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense, ClosedAt: &closedAt}
	temporaltesting.Insert(s.T(), s.DB, closed)

	activityResult, err := s.Env.ExecuteActivity(s.aa.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: active.ID})
	require.NoError(s.T(), err)
	var retrieved lt.RetrieveLedgerAccountResult
	require.NoError(s.T(), activityResult.Get(&retrieved))
	require.NotNil(s.T(), retrieved.LedgerAccount)
	assert.Equal(s.T(), active.ID, retrieved.LedgerAccount.ID)
	assert.Equal(s.T(), currency.Code, retrieved.LedgerAccount.Currency.Code)

	activityResult, err = s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, lt.ListLedgerAccountsParams{ParentID: &root.ID})
	require.NoError(s.T(), err)
	var listed lt.ListLedgerAccountsResult
	require.NoError(s.T(), activityResult.Get(&listed))
	require.Len(s.T(), listed.LedgerAccounts, 1)
	assert.Equal(s.T(), active.ID, listed.LedgerAccounts[0].ID)

	activityResult, err = s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, lt.ListLedgerAccountsParams{IncludeClosed: true})
	require.NoError(s.T(), err)
	listed = lt.ListLedgerAccountsResult{}
	require.NoError(s.T(), activityResult.Get(&listed))
	assert.Len(s.T(), listed.LedgerAccounts, 3)

	_, err = s.Env.ExecuteActivity(s.aa.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: 9999})
	assert.Contains(s.T(), err.Error(), "resource LedgerAccount not found")
}

// TestListLedgerAccountsNameFilter covers NameContains, a case-insensitive
// substring match on the ledger account's own name, the way a caller
// resolving something like "groceries" to a category would.
func (s *UnitTestSuite) TestListLedgerAccountsNameFilter() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)

	groceries := &model.LedgerAccount{Name: "Groceries", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, groceries)
	salary := &model.LedgerAccount{Name: "Salary", CurrencyID: currency.ID, Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, salary)

	tests := map[string]struct {
		nameContains string
		expIDs       []int64
	}{
		"case-insensitive substring match": {
			nameContains: "grocer",
			expIDs:       []int64{groceries.ID},
		},
		"no match returns empty": {
			nameContains: "nonexistent",
			expIDs:       []int64{},
		},
		"blank filter matches everything": {
			nameContains: "",
			expIDs:       []int64{groceries.ID, salary.ID},
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, lt.ListLedgerAccountsParams{NameContains: tc.nameContains})
			require.NoError(s.T(), err)

			var listed lt.ListLedgerAccountsResult
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
// the given kinds, the way the categories facade lists income and expense
// accounts without the asset accounts behind financial accounts.
func (s *UnitTestSuite) TestListLedgerAccountsKindFilter() {
	currency := &model.Currency{Code: "EUR", Name: "Euro", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)

	cash := &model.LedgerAccount{Name: "Cash", CurrencyID: currency.ID, Kind: model.LedgerAccountAsset}
	groceries := &model.LedgerAccount{Name: "Groceries", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	salary := &model.LedgerAccount{Name: "Salary", CurrencyID: currency.ID, Kind: model.LedgerAccountIncome}
	for _, account := range []*model.LedgerAccount{cash, groceries, salary} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	tests := map[string]struct {
		kinds     []model.LedgerAccountKind
		expIDs    []int64
		wantError string
	}{
		"income and expense": {
			kinds:  []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense},
			expIDs: []int64{groceries.ID, salary.ID},
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
			expIDs: []int64{cash.ID, groceries.ID, salary.ID},
		},
		"an unknown kind": {
			kinds:     []model.LedgerAccountKind{"clearly-invalid-kind"},
			wantError: "validation error",
		},
	}
	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.aa.ListLedgerAccounts, lt.ListLedgerAccountsParams{Kinds: tc.kinds})
			if tc.wantError != "" {
				require.ErrorContains(s.T(), err, tc.wantError)
				return
			}
			require.NoError(s.T(), err)

			var listed lt.ListLedgerAccountsResult
			require.NoError(s.T(), activityResult.Get(&listed))
			gotIDs := make([]int64, 0, len(listed.LedgerAccounts))
			for _, a := range listed.LedgerAccounts {
				gotIDs = append(gotIDs, a.ID)
			}
			assert.ElementsMatch(s.T(), tc.expIDs, gotIDs)
		})
	}
}

func (s *UnitTestSuite) TestRenameAndMoveLedgerAccount() {
	currency := &model.Currency{Code: "CAD", Name: "Canadian Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	rootA := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	rootB := &model.LedgerAccount{Name: "Living", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, rootA)
	temporaltesting.Insert(s.T(), s.DB, rootB)
	account := &model.LedgerAccount{Name: "Food", ParentID: &rootA.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, account)

	activityResult, err := s.Env.ExecuteActivity(s.aa.RenameLedgerAccount, lt.RenameLedgerAccountInput{LedgerAccountID: account.ID, Name: "Groceries"})
	require.NoError(s.T(), err)
	var mutation lt.LedgerAccountMutationResult
	require.NoError(s.T(), activityResult.Get(&mutation))
	assert.Equal(s.T(), account.ID, mutation.LedgerAccountID)

	activityResult, err = s.Env.ExecuteActivity(s.aa.MoveLedgerAccount, lt.MoveLedgerAccountInput{LedgerAccountID: account.ID, ParentID: &rootB.ID})
	require.NoError(s.T(), err)
	require.NoError(s.T(), activityResult.Get(&mutation))

	var stored model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	require.Equal(s.T(), "Groceries", stored.Name)
	require.Equal(s.T(), rootB.ID, *stored.ParentID)

	_, err = s.Env.ExecuteActivity(s.aa.MoveLedgerAccount, lt.MoveLedgerAccountInput{LedgerAccountID: rootB.ID, ParentID: &account.ID})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "would create a cycle")
}

func (s *UnitTestSuite) TestMoveLedgerAccountRejectsUsedAccount() {
	currency := &model.Currency{Code: "CHF", Name: "Swiss Franc", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	rootA := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	rootB := &model.LedgerAccount{Name: "Other", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	account := &model.LedgerAccount{Name: "Food", ParentID: &rootA.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, rootA)
	temporaltesting.Insert(s.T(), s.DB, rootB)
	account.ParentID = &rootA.ID
	temporaltesting.Insert(s.T(), s.DB, account)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: account.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit})

	_, err := s.Env.ExecuteActivity(s.aa.MoveLedgerAccount, lt.MoveLedgerAccountInput{LedgerAccountID: account.ID, ParentID: &rootB.ID})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "cannot be moved")
	var stored model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", account.ID).Scan(s.T().Context()))
	assert.Equal(s.T(), rootA.ID, *stored.ParentID)
}

func (s *UnitTestSuite) TestCloseLedgerAccountIsIdempotentAndRejectsActiveChildren() {
	currency := testhelpers.JapaneseYen()
	temporaltesting.Insert(s.T(), s.DB, currency)
	parent := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	child := &model.LedgerAccount{Name: "Food", ParentID: &parent.ID, CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, parent)
	child.ParentID = &parent.ID
	temporaltesting.Insert(s.T(), s.DB, child)

	_, err := s.Env.ExecuteActivity(s.aa.CloseLedgerAccount, lt.CloseLedgerAccountInput{LedgerAccountID: parent.ID, ClosedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "active children")

	_, err = s.Env.ExecuteActivity(s.aa.CloseLedgerAccount, lt.CloseLedgerAccountInput{LedgerAccountID: child.ID, ClosedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.aa.CloseLedgerAccount, lt.CloseLedgerAccountInput{LedgerAccountID: child.ID, ClosedAt: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)})
	require.NoError(s.T(), err)

	var stored model.LedgerAccount
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", child.ID).Scan(s.T().Context()))
	require.NotNil(s.T(), stored.ClosedAt)
	assert.Equal(s.T(), 2026, stored.ClosedAt.Year())
}

func (s *UnitTestSuite) TestDeleteLedgerAccountOnlyAllowsUnusedLeaf() {
	currency := &model.Currency{Code: "AUD", Name: "Australian Dollar", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, currency)
	parent := &model.LedgerAccount{Name: "Expenses", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	child := &model.LedgerAccount{Name: "Food", CurrencyID: currency.ID, Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, parent)
	child.ParentID = &parent.ID
	temporaltesting.Insert(s.T(), s.DB, child)

	_, err := s.Env.ExecuteActivity(s.aa.DeleteLedgerAccount, lt.DeleteLedgerAccountInput{LedgerAccountID: parent.ID})
	require.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "has children")

	_, err = s.Env.ExecuteActivity(s.aa.DeleteLedgerAccount, lt.DeleteLedgerAccountInput{LedgerAccountID: child.ID})
	require.NoError(s.T(), err)

	assert.Zero(s.T(), s.CountWhere[model.LedgerAccount]("id = ?", child.ID))
}

func int64Ptr(value int64) *int64 { return &value }
