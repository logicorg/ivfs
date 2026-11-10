package reporting

import (
	"slices"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestLedgerAccountBalanceAndFinancialAccountBalances() {
	fixture := s.buildReportingFixture(s.T())

	asset := &model.LedgerAccount{FinancialAccountID: &fixture.BankFinancialAccount.ID, CurrencyID: fixture.Currency.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	offset := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: "offset", Kind: model.LedgerAccountExpense}
	temporaltesting.Insert(s.T(), s.DB, asset)
	temporaltesting.Insert(s.T(), s.DB, offset)

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "balance-entry", OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
		{LedgerAccountID: asset.ID, Amount: decimal.NewFromInt(30), Side: model.PostingDebit},
		{LedgerAccountID: offset.ID, Amount: decimal.NewFromInt(30), Side: model.PostingCredit},
	}})
	require.NoError(s.T(), err)
	var postedResult lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	balanceResult, err := s.Env.ExecuteActivity(s.ra.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: asset.ID})
	require.NoError(s.T(), err)
	var balance lt.LedgerAccountBalanceResult
	require.NoError(s.T(), balanceResult.Get(&balance))
	require.True(s.T(), balance.Balance.Equal(decimal.NewFromInt(30)))

	accountsResult, err := s.Env.ExecuteActivity(s.ra.FinancialAccountBalances, lt.FinancialAccountBalancesParams{FinancialAccountID: fixture.BankFinancialAccount.ID})
	require.NoError(s.T(), err)
	var accounts lt.FinancialAccountBalancesResult
	require.NoError(s.T(), accountsResult.Get(&accounts))
	require.Len(s.T(), accounts.Balances, 1)
	require.True(s.T(), accounts.Balances[0].Balance.Equal(decimal.NewFromInt(30)))
	require.Equal(s.T(), asset.Name, accounts.Balances[0].LedgerAccountName)
	require.Equal(s.T(), "EUR", accounts.Balances[0].CurrencyCode)
}

func (s *UnitTestSuite) TestPositionsAtExcludesDraftEntries() {
	fixture := s.buildReportingFixture(s.T())

	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	newMovement := func(units int64) *model.PositionMovement {
		return &model.PositionMovement{FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(units)}
	}

	posted := &model.JournalEntry{OccurredAt: time.Now().UTC()}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, posted,
		[]model.Posting{
			{LedgerAccountID: fixture.Cash.ID, Amount: decimal.NewFromInt(1), Side: model.PostingDebit},
			{LedgerAccountID: fixture.Expense.ID, Amount: decimal.NewFromInt(1), Side: model.PostingCredit},
		},
		[]*model.PositionMovement{newMovement(2)},
	)

	draft := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draft)
	draftMovement := newMovement(100)
	draftMovement.JournalEntryID, draftMovement.OccurredAt = &draft.ID, draft.OccurredAt
	temporaltesting.InsertPositionMovement(s.T(), s.DB, draftMovement)

	activityResult, err := s.Env.ExecuteActivity(s.ra.PositionsAt, lt.PositionsAtParams{FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: asset.ID})
	require.NoError(s.T(), err)
	var result lt.PositionsAtResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.Len(s.T(), result.Positions, 1)
	require.True(s.T(), result.Positions[0].Units.Equal(decimal.NewFromInt(2)))
}

// TestPositionsAt_CountsMovementsWithoutEntry checks that movements that have
// no journal entry are counted, grouped per financial account, and cut off at
// the requested time.
func (s *UnitTestSuite) TestPositionsAt_CountsMovementsWithoutEntry() {
	fixture := s.buildReportingFixture(s.T())

	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)
	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{FinancialAccountID: fixture.BankFinancialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(10), OccurredAt: january})
	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(3), OccurredAt: february})

	tests := map[string]struct {
		at       time.Time
		expected map[int64]int64
	}{
		"now":             {expected: map[int64]int64{fixture.BankFinancialAccount.ID: 10, fixture.BrokerFinancialAccount.ID: 3}},
		"cut off at time": {at: january, expected: map[int64]int64{fixture.BankFinancialAccount.ID: 10}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.PositionsAt, lt.PositionsAtParams{AssetID: asset.ID, At: tc.at})
			require.NoError(s.T(), err)

			var result lt.PositionsAtResult
			require.NoError(s.T(), activityResult.Get(&result))

			units := make(map[int64]int64, len(result.Positions))
			for _, position := range result.Positions {
				units[position.FinancialAccountID] = position.Units.IntPart()
			}
			require.Equal(s.T(), tc.expected, units)
		})
	}
}

func (s *UnitTestSuite) TestListedPostingsAndTrialBalance() {
	fixture := s.buildReportingFixture(s.T())

	debit := fixture.Cash
	credit := fixture.Expense
	postedAt := time.Now().UTC()
	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "query-entry", OccurredAt: postedAt, Postings: []lt.PostingInput{
		{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingDebit},
		{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingCredit},
	}})
	require.NoError(s.T(), err)
	var postedResult lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	listResult, err := s.Env.ExecuteActivity(s.ra.ListPostings, lt.ListPostingsParams{LedgerAccountID: debit.ID})
	require.NoError(s.T(), err)
	var postings lt.ListPostingsResult
	require.NoError(s.T(), listResult.Get(&postings))
	require.Len(s.T(), postings.Postings, 1)
	require.True(s.T(), postings.Postings[0].Amount.Equal(decimal.NewFromInt(7)))

	trialResult, err := s.Env.ExecuteActivity(s.ra.TrialBalanceAt, lt.TrialBalanceAtParams{At: postedAt.Add(time.Second)})
	require.NoError(s.T(), err)
	var trial lt.TrialBalanceAtResult
	require.NoError(s.T(), trialResult.Get(&trial))
	require.Len(s.T(), trial.Rows, 2)
	for _, row := range trial.Rows {
		if row.LedgerAccountID == debit.ID {
			require.True(s.T(), row.Debit.Equal(decimal.NewFromInt(7)))
			require.True(s.T(), row.Balance.Equal(decimal.NewFromInt(7)))
		}
	}
}

func (s *UnitTestSuite) TestLedgerAccountBalance_OnlyCountsPostedEntries() {
	fixture := s.buildReportingFixture(s.T())
	account := fixture.Cash
	offset := fixture.Expense

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "posted-balance", OccurredAt: time.Now().UTC(), Postings: []lt.PostingInput{
		{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(30), Side: model.PostingDebit},
		{LedgerAccountID: offset.ID, Amount: decimal.NewFromInt(30), Side: model.PostingCredit},
	}})
	require.NoError(s.T(), err)
	var postedResult lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	draftEntry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draftEntry)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draftEntry.ID, LineNumber: 1, LedgerAccountID: account.ID, Amount: decimal.NewFromInt(99), Side: model.PostingDebit})

	result, err := s.Env.ExecuteActivity(s.ra.LedgerAccountBalance, lt.LedgerAccountBalanceParams{LedgerAccountID: account.ID})
	require.NoError(s.T(), err)
	var balance lt.LedgerAccountBalanceResult
	require.NoError(s.T(), result.Get(&balance))
	require.True(s.T(), balance.Balance.Equal(decimal.NewFromInt(30)))
	require.Equal(s.T(), account.ID, balance.LedgerAccountID)
	require.Equal(s.T(), fixture.Currency.ID, balance.CurrencyID)
}

func (s *UnitTestSuite) TestTrialBalanceAt_ExcludesDraftPostings() {
	fixture := s.buildReportingFixture(s.T())
	debit := fixture.Cash
	credit := fixture.Expense

	postedAt := time.Now().UTC()
	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{IdempotencyKey: "trial-balance-posted", OccurredAt: postedAt, Postings: []lt.PostingInput{
		{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingDebit},
		{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingCredit},
	}})
	require.NoError(s.T(), err)
	var postedResult lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&postedResult))

	draftEntry := &model.JournalEntry{OccurredAt: postedAt.Add(time.Minute), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draftEntry)
	temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draftEntry.ID, LineNumber: 1, LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(99), Side: model.PostingDebit})

	trialResult, err := s.Env.ExecuteActivity(s.ra.TrialBalanceAt, lt.TrialBalanceAtParams{At: postedAt.Add(2 * time.Minute)})
	require.NoError(s.T(), err)
	var trial lt.TrialBalanceAtResult
	require.NoError(s.T(), trialResult.Get(&trial))

	for _, row := range trial.Rows {
		if row.LedgerAccountID == debit.ID {
			require.True(s.T(), row.Debit.Equal(decimal.NewFromInt(25)))
			require.True(s.T(), row.Balance.Equal(decimal.NewFromInt(25)))
			return
		}
	}
	require.Fail(s.T(), "expected debit row for account")
}

// TestTrialBalanceAt_ReportsAccountsWithNothingCountedAtZero checks that an
// account with no posting counted at the requested time still gets a row, at
// zero, whether it has no postings at all or only ones that do not count yet.
// Posted entries cannot be deleted, so each case holds its own accounts, which
// keeps the cases apart without deleting anything.
func (s *UnitTestSuite) TestTrialBalanceAt_ReportsAccountsWithNothingCountedAtZero() {
	fixture := s.buildReportingFixture(s.T())
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		postings func(account, offset *model.LedgerAccount)
	}{
		"no postings": {
			postings: func(_, _ *model.LedgerAccount) {},
		},
		"only draft postings": {
			postings: func(account, offset *model.LedgerAccount) {
				draft := &model.JournalEntry{OccurredAt: at.AddDate(0, 0, -1), Status: model.JournalEntryDraft}
				temporaltesting.Insert(s.T(), s.DB, draft)
				temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draft.ID, LineNumber: 1, LedgerAccountID: account.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit})
				temporaltesting.Insert(s.T(), s.DB, &model.Posting{JournalEntryID: draft.ID, LineNumber: 2, LedgerAccountID: offset.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit})
			},
		},
		"only postings after the requested time": {
			postings: func(account, offset *model.LedgerAccount) {
				temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{OccurredAt: at.AddDate(0, 0, 1)},
					model.Posting{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					model.Posting{LedgerAccountID: offset.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				)
			},
		},
	}

	for name, tc := range tests {
		account := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: name, Kind: model.LedgerAccountAsset}
		offset := &model.LedgerAccount{CurrencyID: fixture.Currency.ID, Name: name + " offset", Kind: model.LedgerAccountExpense}
		temporaltesting.Insert(s.T(), s.DB, account)
		temporaltesting.Insert(s.T(), s.DB, offset)
		tc.postings(account, offset)

		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.TrialBalanceAt, lt.TrialBalanceAtParams{At: at})
			require.NoError(s.T(), err)
			var trial lt.TrialBalanceAtResult
			require.NoError(s.T(), activityResult.Get(&trial))

			idx := slices.IndexFunc(trial.Rows, func(row lt.TrialBalanceRow) bool { return row.LedgerAccountID == account.ID })
			require.NotEqual(s.T(), -1, idx, "expected a row for the account")
			row := trial.Rows[idx]
			require.Equal(s.T(), fixture.Currency.ID, row.CurrencyID)
			require.True(s.T(), row.Debit.IsZero())
			require.True(s.T(), row.Credit.IsZero())
			require.True(s.T(), row.Balance.IsZero())
		})
	}
}

// TestPositionsAt_CostBasis checks that the cost basis of a position is summed
// per currency, and that only a movement whose missing basis has an effect
// makes it unknown: a split never does. Movements are append-only, so
// each case holds its own asset, which keeps the cases apart without deleting
// anything.
func (s *UnitTestSuite) TestPositionsAt_CostBasis() {
	fixture := s.buildReportingFixture(s.T())
	usd := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, usd)

	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	movement := func(asset *model.Asset, units int64, day int) *model.PositionMovement {
		return &model.PositionMovement{
			FinancialAccountID: fixture.BrokerFinancialAccount.ID, AssetID: asset.ID,
			UnitsDelta: decimal.NewFromInt(units), OccurredAt: january.AddDate(0, 0, day),
		}
	}
	withBasis := func(m *model.PositionMovement, amount int64, currency *model.Currency) *model.PositionMovement {
		m.CostBasisAmount, m.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(amount)), &currency.ID
		return m
	}

	tests := map[string]struct {
		movements func(asset *model.Asset)
		at        time.Time
		units     int64
		// costBasis is the expected amount by currency code.
		costBasis map[string]int64
		unknown   bool
		// average is the expected average cost by currency code, and nil when
		// there is none.
		average map[string]int64
	}{
		"known basis summed in one currency": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 5, 1), 50, fixture.Currency))
			},
			units: 15, costBasis: map[string]int64{"EUR": 150}, average: map[string]int64{"EUR": 10},
		},
		"a sale without a basis removes the average cost": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 1), 200, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, movement(asset, -5, 2))
			},
			units: 15, costBasis: map[string]int64{"EUR": 225}, average: map[string]int64{"EUR": 15},
		},
		"a sale is replayed after the buys dated before it": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, movement(asset, -5, 2))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
			},
			units: 5, costBasis: map[string]int64{"EUR": 50}, average: map[string]int64{"EUR": 10},
		},
		// A movement written again with its entry has a later id and keeps when
		// it was recorded (ADR 0017), so the id must not decide the order.
		"a sale is replayed after the buy of the same instant recorded before it": {
			movements: func(asset *model.Asset) {
				date, datePlus1h := january.AddDate(0, 0, 10), january.AddDate(0, 0, 10).Add(time.Hour)
				sale := movement(asset, -5, 0)
				sale.CreatedAt = datePlus1h
				temporaltesting.InsertPositionMovement(s.T(), s.DB, sale)
				buy := withBasis(movement(asset, 10, 0), 100, fixture.Currency)
				buy.CreatedAt = date
				temporaltesting.InsertPositionMovement(s.T(), s.DB, buy)
			},
			units: 5, costBasis: map[string]int64{"EUR": 50}, average: map[string]int64{"EUR": 10},
		},
		"a sale after At removes nothing": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, movement(asset, -5, 2))
			},
			at: january.AddDate(0, 0, 1), units: 10, costBasis: map[string]int64{"EUR": 100}, average: map[string]int64{"EUR": 10},
		},
		"basis kept apart by currency": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 5, 1), 50, usd))
			},
			units: 15, costBasis: map[string]int64{"EUR": 100, "USD": 50},
		},
		"a movement without a basis makes it unknown": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				temporaltesting.InsertPositionMovement(s.T(), s.DB, movement(asset, 5, 1))
			},
			units: 15, costBasis: map[string]int64{"EUR": 100}, unknown: true,
		},
		"a split without a basis does not": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				split := movement(asset, 10, 1)
				split.Kind = model.PositionMovementSplit
				temporaltesting.InsertPositionMovement(s.T(), s.DB, split)
			},
			units: 20, costBasis: map[string]int64{"EUR": 100}, average: map[string]int64{"EUR": 5},
		},
		"a draft entry's movement without a basis does not": {
			movements: func(asset *model.Asset) {
				temporaltesting.InsertPositionMovement(s.T(), s.DB, withBasis(movement(asset, 10, 0), 100, fixture.Currency))
				draft := &model.JournalEntry{OccurredAt: january.AddDate(0, 0, 1), Status: model.JournalEntryDraft}
				temporaltesting.Insert(s.T(), s.DB, draft)
				drafted := movement(asset, 5, 1)
				drafted.JournalEntryID = &draft.ID
				temporaltesting.InsertPositionMovement(s.T(), s.DB, drafted)
			},
			units: 10, costBasis: map[string]int64{"EUR": 100}, average: map[string]int64{"EUR": 10},
		},
	}

	codes := map[int64]string{fixture.Currency.ID: fixture.Currency.Code, usd.ID: usd.Code}
	for name, tc := range tests {
		asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
		temporaltesting.Insert(s.T(), s.DB, asset)
		tc.movements(asset)

		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.ra.PositionsAt, lt.PositionsAtParams{AssetID: asset.ID, At: tc.at})
			require.NoError(s.T(), err)
			var result lt.PositionsAtResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Len(s.T(), result.Positions, 1)

			position := result.Positions[0]
			require.Equal(s.T(), tc.units, position.Units.IntPart())
			costBasis := make(map[string]int64, len(position.CostBasis))
			for _, basis := range position.CostBasis {
				costBasis[codes[basis.CurrencyID]] = basis.Amount.IntPart()
			}
			require.Equal(s.T(), tc.costBasis, costBasis)
			require.Equal(s.T(), tc.unknown, position.UnknownCostBasis)
			var average map[string]int64
			if position.AverageCost != nil {
				average = map[string]int64{codes[position.AverageCost.CurrencyID]: position.AverageCost.Amount.IntPart()}
			}
			require.Equal(s.T(), tc.average, average)
		})
	}
}
