package accounts

import (
	"errors"
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// listAccountsFixtures is what the two list activities return: Revolut
// Personal with a Cash account in EUR and one in USD, a closed N26 account
// whose EUR Cash account is still open, and two categories that belong to no
// financial account.
type listAccountsFixtures struct {
	financialAccounts []model.FinancialAccount
	ledgerAccounts    []model.LedgerAccount

	revolutEUR, revolutUSD, n26EUR, groceries, salary lt.AccountSummary
}

func newListAccountsFixtures() listAccountsFixtures {
	closedAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	euro := &model.Currency{ID: 1, Code: "EUR"}
	dollar := &model.Currency{ID: 2, Code: "USD"}
	revolut := &model.Institution{ID: 1, Name: "Revolut"}
	n26 := &model.Institution{ID: 2, Name: "N26"}

	f := listAccountsFixtures{
		financialAccounts: []model.FinancialAccount{
			{ID: 10, InstitutionID: revolut.ID, Name: "Revolut Personal", Institution: revolut},
			{ID: 20, InstitutionID: n26.ID, Name: "N26 Old", Institution: n26, ClosedAt: &closedAt},
		},
		ledgerAccounts: []model.LedgerAccount{
			{ID: 100, FinancialAccountID: ptr.Int64(10), CurrencyID: euro.ID, Currency: euro, Name: "Cash", Kind: model.LedgerAccountAsset},
			{ID: 101, FinancialAccountID: ptr.Int64(10), CurrencyID: dollar.ID, Currency: dollar, Name: "Cash", Kind: model.LedgerAccountAsset},
			{ID: 200, FinancialAccountID: ptr.Int64(20), CurrencyID: euro.ID, Currency: euro, Name: "Cash", Kind: model.LedgerAccountAsset},
			{ID: 300, CurrencyID: euro.ID, Currency: euro, Name: "Groceries", Kind: model.LedgerAccountExpense},
			{ID: 301, CurrencyID: euro.ID, Currency: euro, Name: "Salary", Kind: model.LedgerAccountIncome},
		},
	}
	f.revolutEUR = lt.AccountSummary{
		LedgerAccountID: 100, LedgerAccountName: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1, CurrencyCode: "EUR",
		FinancialAccountID: ptr.Int64(10), FinancialAccountName: "Revolut Personal", InstitutionName: "Revolut",
	}
	f.revolutUSD = lt.AccountSummary{
		LedgerAccountID: 101, LedgerAccountName: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 2, CurrencyCode: "USD",
		FinancialAccountID: ptr.Int64(10), FinancialAccountName: "Revolut Personal", InstitutionName: "Revolut",
	}
	f.n26EUR = lt.AccountSummary{
		LedgerAccountID: 200, LedgerAccountName: "Cash", Kind: model.LedgerAccountAsset, CurrencyID: 1, CurrencyCode: "EUR",
		FinancialAccountID: ptr.Int64(20), FinancialAccountName: "N26 Old", InstitutionName: "N26", Closed: true,
	}
	f.groceries = lt.AccountSummary{LedgerAccountID: 300, LedgerAccountName: "Groceries", Kind: model.LedgerAccountExpense, CurrencyID: 1, CurrencyCode: "EUR"}
	f.salary = lt.AccountSummary{LedgerAccountID: 301, LedgerAccountName: "Salary", Kind: model.LedgerAccountIncome, CurrencyID: 1, CurrencyCode: "EUR"}
	return f
}

// TestListAccounts checks that each ledger account is listed with the names of
// its financial account and institution, that the kinds and closed filters
// are passed to ListLedgerAccounts, and that the search, currency, and scope
// filters, and the closed state of a financial account, are applied to the
// joined rows.
func (s *WorkflowUnitTestSuite) TestListAccounts() {
	aa := activity.Activities{}
	f := newListAccountsFixtures()

	tests := map[string]struct {
		params lt.ListAccountsParams
		// ledgerParams is what ListLedgerAccounts is expected to be called
		// with, and ledgerAccounts what it returns, all of them if nil.
		ledgerParams   lt.ListLedgerAccountsParams
		ledgerAccounts []model.LedgerAccount
		want           []lt.AccountSummary
	}{
		"every open account": {
			want: []lt.AccountSummary{f.revolutEUR, f.revolutUSD, f.groceries, f.salary},
		},
		"closed accounts, and those of a closed financial account": {
			params:       lt.ListAccountsParams{IncludeClosed: true},
			ledgerParams: lt.ListLedgerAccountsParams{IncludeClosed: true},
			want:         []lt.AccountSummary{f.revolutEUR, f.revolutUSD, f.n26EUR, f.groceries, f.salary},
		},
		"kinds are filtered by ListLedgerAccounts": {
			params:         lt.ListAccountsParams{Kinds: []model.LedgerAccountKind{model.LedgerAccountExpense}},
			ledgerParams:   lt.ListLedgerAccountsParams{Kinds: []model.LedgerAccountKind{model.LedgerAccountExpense}},
			ledgerAccounts: f.ledgerAccounts[3:4],
			want:           []lt.AccountSummary{f.groceries},
		},
		"search words across the institution's name and the currency code": {
			params: lt.ListAccountsParams{Search: "revolut usd"},
			want:   []lt.AccountSummary{f.revolutUSD},
		},
		"search by the ledger account's name, in another casing": {
			params: lt.ListAccountsParams{Search: "GROCER"},
			want:   []lt.AccountSummary{f.groceries},
		},
		"search by the financial account's name": {
			params: lt.ListAccountsParams{Search: "personal"},
			want:   []lt.AccountSummary{f.revolutEUR, f.revolutUSD},
		},
		"search that matches nothing": {
			params: lt.ListAccountsParams{Search: "revolut gbp"},
			want:   []lt.AccountSummary{},
		},
		"currency code, in another casing": {
			params: lt.ListAccountsParams{CurrencyCode: "usd"},
			want:   []lt.AccountSummary{f.revolutUSD},
		},
		"accounts that belong to a financial account": {
			params: lt.ListAccountsParams{Scope: lt.AccountScopeFinancial},
			want:   []lt.AccountSummary{f.revolutEUR, f.revolutUSD},
		},
		"accounts that belong to none": {
			params: lt.ListAccountsParams{Scope: lt.AccountScopeUnlinked},
			want:   []lt.AccountSummary{f.groceries, f.salary},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			ledgerAccounts := tc.ledgerAccounts
			if ledgerAccounts == nil {
				ledgerAccounts = f.ledgerAccounts
			}

			env := s.NewTestWorkflowEnvironment()
			env.OnActivity(aa.ListLedgerAccounts, mock.Anything, tc.ledgerParams).
				Return(&lt.ListLedgerAccountsResult{LedgerAccounts: ledgerAccounts}, nil)
			env.OnActivity(aa.ListFinancialAccounts, mock.Anything, lt.ListFinancialAccountsParams{IncludeClosed: true}).
				Return(&lt.ListFinancialAccountsResult{FinancialAccounts: f.financialAccounts}, nil)
			env.ExecuteWorkflow(ListAccounts, tc.params)

			require.NoError(s.T(), env.GetWorkflowError())

			var listed *lt.ListAccountsResult
			require.NoError(s.T(), env.GetWorkflowResult(&listed))
			require.Equal(s.T(), tc.want, listed.Accounts)
		})
	}
}

// TestListAccounts_Errors checks that an unknown scope is rejected before any
// activity runs, that a ledger account whose financial account was not listed
// fails the workflow rather than losing its names, that one without its
// currency panics, which the test environment reports as the workflow's
// error, and that an activity's error is returned.
func (s *WorkflowUnitTestSuite) TestListAccounts_Errors() {
	aa := activity.Activities{}
	f := newListAccountsFixtures()
	orphan := model.LedgerAccount{ID: 400, FinancialAccountID: ptr.Int64(30), CurrencyID: 1, Currency: &model.Currency{ID: 1, Code: "EUR"}, Name: "Cash", Kind: model.LedgerAccountAsset}

	tests := map[string]struct {
		params lt.ListAccountsParams
		// mock sets up the activities the case reaches, if any.
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"an unknown scope": {
			params:    lt.ListAccountsParams{Scope: "everything"},
			mock:      func(*testsuite.TestWorkflowEnvironment) {},
			wantError: "unknown account scope",
		},
		"a financial account that was not listed": {
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(aa.ListLedgerAccounts, mock.Anything, mock.Anything).
					Return(&lt.ListLedgerAccountsResult{LedgerAccounts: append(f.ledgerAccounts, orphan)}, nil)
				env.OnActivity(aa.ListFinancialAccounts, mock.Anything, mock.Anything).
					Return(&lt.ListFinancialAccountsResult{FinancialAccounts: f.financialAccounts}, nil)
			},
			wantError: "financial account 30 of ledger account 400 was not listed",
		},
		"a ledger account without its currency": {
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				noCurrency := f.ledgerAccounts[0]
				noCurrency.Currency = nil
				env.OnActivity(aa.ListLedgerAccounts, mock.Anything, mock.Anything).
					Return(&lt.ListLedgerAccountsResult{LedgerAccounts: []model.LedgerAccount{noCurrency}}, nil)
				env.OnActivity(aa.ListFinancialAccounts, mock.Anything, mock.Anything).
					Return(&lt.ListFinancialAccountsResult{FinancialAccounts: f.financialAccounts}, nil)
			},
			wantError: "ledger account 100 has no currency loaded",
		},
		"listing the ledger accounts fails": {
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(aa.ListLedgerAccounts, mock.Anything, mock.Anything).Return(nil, errors.New("database down"))
			},
			wantError: "list ledger accounts",
		},
		"listing the financial accounts fails": {
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnActivity(aa.ListLedgerAccounts, mock.Anything, mock.Anything).
					Return(&lt.ListLedgerAccountsResult{LedgerAccounts: f.ledgerAccounts}, nil)
				env.OnActivity(aa.ListFinancialAccounts, mock.Anything, mock.Anything).Return(nil, errors.New("database down"))
			},
			wantError: "list financial accounts",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			tc.mock(env)
			env.ExecuteWorkflow(ListAccounts, tc.params)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
