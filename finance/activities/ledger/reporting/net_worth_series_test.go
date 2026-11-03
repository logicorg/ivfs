package reporting

import (
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// netWorthRow is a balances or a units row with its figures as text, so a
// case can spell them out and they compare at any scale.
type netWorthRow struct {
	date    string
	id      int64
	figures string
}

// Test_NetWorthSeries checks the sparse series: a row on From for what was
// carried in, then rows only on the days a figure changes, in UTC days, with
// drafts and the brokerage "Investments" balance left out. The history is
// seeded once and only read by the cases, since a posted entry cannot be
// deleted.
func (s *UnitTestSuite) Test_NetWorthSeries() {
	fixture := s.buildReportingFixture(s.T())
	euro := fixture.Currency
	dollar := &model.Currency{Code: "USD", Name: "US dollar", Symbol: "$", Precision: 2}
	temporaltesting.Insert(s.T(), s.DB, dollar)

	february20 := time.Date(2026, 2, 20, 0, 0, 0, 0, time.UTC)
	salary := &model.LedgerAccount{CurrencyID: euro.ID, Name: "Salary", Kind: model.LedgerAccountIncome}
	card := &model.LedgerAccount{CurrencyID: euro.ID, Name: "Credit card", Kind: model.LedgerAccountLiability}
	brokerCash := &model.LedgerAccount{CurrencyID: euro.ID, FinancialAccountID: &fixture.BrokerFinancialAccount.ID, Name: model.BrokerageCashBalance, Kind: model.LedgerAccountAsset, ClosedAt: &february20}
	investments := &model.LedgerAccount{CurrencyID: euro.ID, FinancialAccountID: &fixture.BrokerFinancialAccount.ID, Name: model.BrokerageInvestmentsBalance, Kind: model.LedgerAccountAsset}
	dollarCash := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Revolut USD", Kind: model.LedgerAccountAsset}
	dollarIncome := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Freelance", Kind: model.LedgerAccountIncome}
	travel := &model.LedgerAccount{CurrencyID: dollar.ID, Name: "Travel", Kind: model.LedgerAccountExpense}
	for _, account := range []*model.LedgerAccount{salary, card, brokerCash, investments, dollarCash, dollarIncome, travel} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}
	apple := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	microsoft := &model.Asset{Name: "MSFT", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, apple)
	temporaltesting.Insert(s.T(), s.DB, microsoft)

	day := func(month time.Month, day int) time.Time { return time.Date(2026, month, day, 0, 0, 0, 0, time.UTC) }
	postings := func(debit, credit *model.LedgerAccount, amount int64) []model.Posting {
		return []model.Posting{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit},
		}
	}
	post := func(at time.Time, debit, credit *model.LedgerAccount, amount int64) {
		temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: at}, postings(debit, credit, amount)...)
	}
	adjust := func(at time.Time, account *model.FinancialAccount, asset *model.Asset, units string) {
		temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{
			FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.RequireFromString(units), OccurredAt: at,
		})
	}

	post(time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC), fixture.Cash, salary, 1000)
	post(time.Date(2025, 12, 31, 23, 30, 0, 0, time.UTC), fixture.Expense, card, 50)
	post(day(1, 5), dollarCash, dollarIncome, 200)
	post(day(1, 8), brokerCash, fixture.Cash, 500)
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, &model.JournalEntry{OccurredAt: day(1, 10)},
		postings(investments, brokerCash, 300),
		[]*model.PositionMovement{{FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: apple.ID, UnitsDelta: decimal.NewFromInt(3)}})
	adjust(day(1, 20), fixture.BankFinancialAccount, apple, "2.5")
	adjust(day(1, 25), fixture.BrokerFinancialAccount, microsoft, "-1")
	post(day(2, 1), card, fixture.Cash, 50)
	post(day(2, 10), travel, dollarCash, 200)
	adjust(day(2, 15), fixture.BrokerFinancialAccount, microsoft, "1")

	draft := &model.JournalEntry{OccurredAt: day(1, 15), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draft)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draft.ID, LineNumber: 1, LedgerAccountID: fixture.Cash.ID, Amount: decimal.NewFromInt(999), Side: model.PostingDebit})
	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{
		JournalEntryID: &draft.ID, FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: apple.ID, UnitsDelta: decimal.NewFromInt(100), OccurredAt: draft.OccurredAt,
	})

	tests := map[string]struct {
		from, to     time.Time
		wantBalances []netWorthRow
		wantUnits    []netWorthRow
	}{
		"the carry-in on From, then the days that change, with the times of day ignored": {
			from: day(1, 1).Add(15 * time.Hour), to: day(1, 31).Add(8 * time.Hour),
			wantBalances: []netWorthRow{
				{"2026-01-01", euro.ID, "1000/50"},
				{"2026-01-05", dollar.ID, "200/0"},
				{"2026-01-10", euro.ID, "700/50"},
			},
			wantUnits: []netWorthRow{
				{"2026-01-10", apple.ID, "3"},
				{"2026-01-20", apple.ID, "5.5"},
				{"2026-01-25", microsoft.ID, "-1"},
			},
		},
		"a From mid-range carries every currency and asset in": {
			from: day(1, 12), to: day(2, 28),
			wantBalances: []netWorthRow{
				{"2026-01-12", euro.ID, "700/50"},
				{"2026-01-12", dollar.ID, "200/0"},
				{"2026-02-01", euro.ID, "650/0"},
				{"2026-02-10", dollar.ID, "0/0"},
			},
			wantUnits: []netWorthRow{
				{"2026-01-12", apple.ID, "3"},
				{"2026-01-20", apple.ID, "5.5"},
				{"2026-01-25", microsoft.ID, "-1"},
				{"2026-02-15", microsoft.ID, "0"},
			},
		},
		"what is back at zero is left out of From": {
			from: day(3, 1), to: day(3, 31),
			wantBalances: []netWorthRow{{"2026-03-01", euro.ID, "650/0"}},
			wantUnits:    []netWorthRow{{"2026-03-01", apple.ID, "5.5"}},
		},
		"a day is a UTC day, its last half hour included": {
			from: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC), to: time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC),
			wantBalances: []netWorthRow{{"2025-12-31", euro.ID, "1000/50"}},
			wantUnits:    []netWorthRow{},
		},
		"To leaves out the next day": {
			from: time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC), to: time.Date(2025, 12, 30, 0, 0, 0, 0, time.UTC),
			wantBalances: []netWorthRow{{"2025-12-20", euro.ID, "1000/0"}},
			wantUnits:    []netWorthRow{},
		},
		"a range before any entry": {
			from: time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), to: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
			wantBalances: []netWorthRow{},
			wantUnits:    []netWorthRow{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.NetWorthSeries, lt.NetWorthSeriesParams{From: tc.from, To: tc.to})
			require.NoError(s.T(), err)

			var got lt.NetWorthSeriesResult
			require.NoError(s.T(), activityResult.Get(&got))
			gotBalances := make([]netWorthRow, len(got.Balances))
			for n, row := range got.Balances {
				gotBalances[n] = netWorthRow{row.Date.Format(time.DateOnly), row.CurrencyID, fmt.Sprintf("%s/%s", row.Assets, row.Liabilities)}
				require.Equal(s.T(), row.Date, row.Date.Truncate(24*time.Hour), "a date is midnight UTC")
			}
			gotUnits := make([]netWorthRow, len(got.Units))
			for n, row := range got.Units {
				gotUnits[n] = netWorthRow{row.Date.Format(time.DateOnly), row.AssetID, row.Units.String()}
			}
			require.Equal(s.T(), tc.wantBalances, gotBalances)
			require.Equal(s.T(), tc.wantUnits, gotUnits)
		})
	}
}

func (s *UnitTestSuite) Test_NetWorthSeries_Errors() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		params    lt.NetWorthSeriesParams
		wantError string
	}{
		"no range":        {params: lt.NetWorthSeriesParams{}, wantError: "missing required parameter"},
		"no end":          {params: lt.NetWorthSeriesParams{From: january}, wantError: "missing required parameter"},
		"a backwards one": {params: lt.NetWorthSeriesParams{From: january, To: january.AddDate(0, 0, -1)}, wantError: "invalid parameter"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ra.NetWorthSeries, tc.params)
			require.ErrorContains(s.T(), err, tc.wantError)
		})
	}
}

// Test_NetWorthSeries_TooManyChanges checks that a series past the row limit
// is refused rather than sent over Temporal's payload limit.
func (s *UnitTestSuite) Test_NetWorthSeries_TooManyChanges() {
	fixture := s.buildReportingFixture(s.T())
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	// One adjustment a day, one more than the limit; inserted in SQL, since
	// ten thousand fixtures one by one would dominate the suite's run time.
	start := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := s.DB.NewRaw(`INSERT INTO asset.position_movement (kind, financial_account_id, asset_id, units_delta, occurred_at, idempotency_key, line_number)
		SELECT ?, ?, ?, 1, ?::timestamptz + make_interval(days => n), 'too-many-' || n, 1 FROM generate_series(0, ?) AS n`,
		model.PositionMovementAdjustment, fixture.BrokerFinancialAccount.ID, asset.ID, start, lt.NetWorthSeriesMaxRows).Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.ra.NetWorthSeries, lt.NetWorthSeriesParams{From: start, To: start.AddDate(30, 0, 0)})
	require.ErrorContains(s.T(), err, "the range holds 10001 changes")
}
