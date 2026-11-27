package reporting

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_CategoryTotals checks that the postings to income and expense accounts
// are totalled by category and currency, in the account's natural direction,
// over the posted entries of a half-open period, and that those without a
// category make a total of their own. The entries are seeded once and only
// read by the cases, since a posted entry cannot be deleted.
func (s *UnitTestSuite) Test_CategoryTotals() {
	fixture := s.buildReportingFixture(s.T())
	dollar := &model.Currency{Code: "USD", Name: "US dollar", Symbol: "$", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, dollar)
	euroID := fixture.Currency.ID

	cash := &model.LedgerAccount{CurrencyID: euroID, Name: "Revolut EUR", Kind: model.LedgerAccountAsset}
	dollarCash := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Revolut USD", Kind: model.LedgerAccountAsset}
	income := &model.LedgerAccount{CurrencyID: euroID, Name: lt.IncomeAccountName, Kind: model.LedgerAccountIncome}
	expenses := &model.LedgerAccount{CurrencyID: euroID, Name: lt.ExpensesAccountName, Kind: model.LedgerAccountExpense}
	dollarExpenses := &model.LedgerAccount{CurrencyID: dollar.ID, Name: lt.ExpensesAccountName, Kind: model.LedgerAccountExpense}
	for _, account := range []*model.LedgerAccount{cash, dollarCash, income, expenses, dollarExpenses} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}
	// The categories are inserted in the order the totals list them.
	salary := &model.Category{Name: "Salary", Kind: model.LedgerAccountIncome}
	groceries := &model.Category{Name: "Groceries", Kind: model.LedgerAccountExpense}
	travel := &model.Category{Name: "Travel", Kind: model.LedgerAccountExpense}
	for _, category := range []*model.Category{salary, groceries, travel} {
		temporaltesting.Insert(s.T(), s.DB, category)
	}

	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)
	march := february.AddDate(0, 1, 0)
	// post books amount from credit to debit; the category, nil for none, is
	// of the posting to the income or expense account.
	post := func(at time.Time, debit, credit *model.LedgerAccount, category *model.Category, amount int64) {
		postings := []model.Posting{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit},
		}
		if category != nil {
			categorized := 0
			if credit.Kind == category.Kind {
				categorized = 1
			}
			postings[categorized].CategoryID = &category.ID
		}
		temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: at}, postings...)
	}
	post(january, cash, income, salary, 2000)
	post(january.AddDate(0, 0, 10), expenses, cash, groceries, 120)
	post(january.AddDate(0, 0, 12), expenses, cash, nil, 15)
	post(january.AddDate(0, 0, 20), dollarExpenses, dollarCash, travel, 300)
	post(january.AddDate(0, 0, 21), dollarExpenses, dollarCash, groceries, 20)
	post(february, expenses, cash, groceries, 80)
	post(february.AddDate(0, 0, 5), cash, expenses, groceries, 100)

	draft := &model.JournalEntry{OccurredAt: january.AddDate(0, 0, 15), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draft)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draft.ID, LineNumber: 1, LedgerAccountID: expenses.ID, CategoryID: &groceries.ID, Amount: decimal.NewFromInt(999), Side: model.PostingDebit})

	// total is a row wanted: a category, nil for the uncategorized, in euros
	// unless dollars is set.
	type total struct {
		category *model.Category
		dollars  bool
		amount   int64
	}
	tests := map[string]struct {
		from, to time.Time
		want     []total
	}{
		"a month, with its start included and drafts left out": {
			from: january, to: february,
			want: []total{{salary, false, 2000}, {groceries, false, 120}, {groceries, true, 20}, {travel, true, 300}, {nil, false, 15}},
		},
		"a refund larger than the spending": {
			from: february, to: march,
			want: []total{{groceries, false, -20}},
		},
		"several months": {
			from: january, to: march,
			want: []total{{salary, false, 2000}, {groceries, false, 100}, {groceries, true, 20}, {travel, true, 300}, {nil, false, 15}},
		},
		"its end left out": {
			from: january.AddDate(0, 0, 5), to: february,
			want: []total{{groceries, false, 120}, {groceries, true, 20}, {travel, true, 300}, {nil, false, 15}},
		},
		"a period without entries": {
			from: march, to: march.AddDate(0, 1, 0),
			want: []total{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.CategoryTotals, rpt.CategoryTotalsParams{From: tc.from, To: tc.to})
			require.NoError(s.T(), err)

			var got rpt.CategoryTotalsResult
			require.NoError(s.T(), activityResult.Get(&got))
			require.Len(s.T(), got.Totals, len(tc.want))
			for i, want := range tc.want {
				require.Truef(s.T(), got.Totals[i].Total.Equal(decimal.NewFromInt(want.amount)), "total %d is %s, want %d", i, got.Totals[i].Total, want.amount)
				got.Totals[i].Total = decimal.Decimal{}
				wantTotal := rpt.CategoryTotal{Kind: model.LedgerAccountExpense, CurrencyID: euroID, CurrencyCode: "EUR"}
				if want.category != nil {
					wantTotal.CategoryID, wantTotal.Kind = &want.category.ID, want.category.Kind
				}
				if want.dollars {
					wantTotal.CurrencyID, wantTotal.CurrencyCode = dollar.ID, "USD"
				}
				require.Equal(s.T(), wantTotal, got.Totals[i])
			}
		})
	}
}

func (s *UnitTestSuite) Test_CategoryTotals_Errors() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		params    rpt.CategoryTotalsParams
		wantError string
	}{
		"no period":     {params: rpt.CategoryTotalsParams{}, wantError: "missing required parameter"},
		"an empty one":  {params: rpt.CategoryTotalsParams{From: january, To: january}, wantError: "invalid parameter"},
		"one backwards": {params: rpt.CategoryTotalsParams{From: january, To: january.AddDate(0, 0, -1)}, wantError: "invalid parameter"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ra.CategoryTotals, tc.params)
			require.ErrorContains(s.T(), err, tc.wantError)
		})
	}
}
