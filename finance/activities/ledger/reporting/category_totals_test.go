package reporting

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_CategoryTotals checks that each income and expense account is totalled
// in its natural direction over the posted entries of a half-open period. The
// entries are seeded once and only read by the cases, since a posted entry
// cannot be deleted.
func (s *UnitTestSuite) Test_CategoryTotals() {
	fixture := s.buildReportingFixture(s.T())
	dollar := &model.Currency{Code: "USD", Name: "US dollar", Symbol: "$", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, dollar)

	cash := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Revolut EUR", Kind: model.LedgerAccountAsset}
	dollarCash := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Revolut USD", Kind: model.LedgerAccountAsset}
	salary := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Salary", Kind: model.LedgerAccountIncome}
	groceries := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "Groceries", Kind: model.LedgerAccountExpense}
	travel := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Travel", Kind: model.LedgerAccountExpense}
	for _, account := range []*model.LedgerAccount{cash, dollarCash, salary, groceries, travel} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)
	march := february.AddDate(0, 1, 0)
	post := func(at time.Time, debit, credit *model.LedgerAccount, amount int64) {
		temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: at},
			model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit},
			model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit},
		)
	}
	post(january, cash, salary, 2000)
	post(january.AddDate(0, 0, 10), groceries, cash, 120)
	post(january.AddDate(0, 0, 20), travel, dollarCash, 300)
	post(february, groceries, cash, 80)
	post(february.AddDate(0, 0, 5), cash, groceries, 100)

	draft := &model.JournalEntry{OccurredAt: january.AddDate(0, 0, 15), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draft)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draft.ID, LineNumber: 1, LedgerAccountID: groceries.ID, Amount: decimal.NewFromInt(999), Side: model.PostingDebit})

	type total struct {
		account *model.LedgerAccount
		amount  int64
	}
	tests := map[string]struct {
		from, to time.Time
		want     []total
	}{
		"a month, with its start included and drafts left out": {
			from: january, to: february,
			want: []total{{salary, 2000}, {groceries, 120}, {travel, 300}},
		},
		"a refund larger than the spending": {
			from: february, to: march,
			want: []total{{groceries, -20}},
		},
		"several months": {
			from: january, to: march,
			want: []total{{salary, 2000}, {groceries, 100}, {travel, 300}},
		},
		"its end left out": {
			from: january.AddDate(0, 0, 5), to: february,
			want: []total{{groceries, 120}, {travel, 300}},
		},
		"a period without entries": {
			from: march, to: march.AddDate(0, 1, 0),
			want: []total{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.CategoryTotals, lt.CategoryTotalsParams{From: tc.from, To: tc.to})
			require.NoError(s.T(), err)

			var got lt.CategoryTotalsResult
			require.NoError(s.T(), activityResult.Get(&got))
			require.Len(s.T(), got.Totals, len(tc.want))
			for i, want := range tc.want {
				require.Truef(s.T(), got.Totals[i].Total.Equal(decimal.NewFromInt(want.amount)), "%s total %s, want %d", want.account.Name, got.Totals[i].Total, want.amount)
				got.Totals[i].Total = decimal.Decimal{}
				code := map[int64]string{fixture.Currency.ID: "EUR", dollar.ID: "USD"}[want.account.CurrencyID]
				require.Equal(s.T(), lt.CategoryTotal{
					LedgerAccountID: want.account.ID,
					Kind:            want.account.Kind,
					CurrencyID:      want.account.CurrencyID,
					CurrencyCode:    code,
				}, got.Totals[i])
			}
		})
	}
}

func (s *UnitTestSuite) Test_CategoryTotals_Errors() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		params    lt.CategoryTotalsParams
		wantError string
	}{
		"no period":     {params: lt.CategoryTotalsParams{}, wantError: "missing required parameter"},
		"an empty one":  {params: lt.CategoryTotalsParams{From: january, To: january}, wantError: "invalid parameter"},
		"one backwards": {params: lt.CategoryTotalsParams{From: january, To: january.AddDate(0, 0, -1)}, wantError: "invalid parameter"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ra.CategoryTotals, tc.params)
			require.ErrorContains(s.T(), err, tc.wantError)
		})
	}
}
