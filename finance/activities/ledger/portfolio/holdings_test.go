package portfolio

import (
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var holdingsBaseDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// trade builds a buy for positive units and a sale for negative ones. The
// entry only has to balance, so both move 100 between the same two accounts.
func (s *UnitTestSuite) trade(key string, asset *model.Asset, units int64, day int) lt.RegisterInvestmentTradeInput {
	debit, credit := s.offset, s.cash
	if units < 0 {
		debit, credit = s.cash, s.offset
	}

	return lt.RegisterInvestmentTradeInput{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            asset.ID,
		UnitsDelta:         decimal.NewFromInt(units),
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: key,
			OccurredAt:     holdingsBaseDay.AddDate(0, 0, day),
			Postings: []lt.PostingInput{
				{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
				{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
			},
		},
	}
}

// negativeAt is the warning a case expects: the units held at the first point
// where the holding is negative, and the day of that point.
type negativeAt struct {
	units int64
	day   int
}

// expectedWarning is a warning for the position of an account.
type expectedWarning struct {
	account *model.FinancialAccount
	at      negativeAt
}

// requireWarnings checks that the warnings are exactly one per expected
// position, in the order given, each at the first negative point of its
// position.
func (s *UnitTestSuite) requireWarnings(got []lt.NegativeHolding, asset *model.Asset, want ...expectedWarning) {
	s.T().Helper()

	require.Len(s.T(), got, len(want))
	for n, w := range want {
		require.Equal(s.T(), w.account.ID, got[n].FinancialAccountID)
		require.Equal(s.T(), w.account.Name, got[n].FinancialAccountName)
		require.Equal(s.T(), asset.ID, got[n].AssetID)
		require.Equal(s.T(), asset.Name, got[n].AssetName)
		require.True(s.T(), got[n].Units.Equal(decimal.NewFromInt(w.at.units)), "holds %s units", got[n].Units)
		require.True(s.T(), got[n].At.Equal(movementDay.AddDate(0, 0, w.at.day)), "negative at %s", got[n].At)
	}
}

// Test_RegisterInvestmentTrade_Holdings checks that a trade is written whatever
// it leaves held, so a sale can come before its purchase, and that a position
// that is negative at some point in time is returned as a warning naming the
// first such point.
func (s *UnitTestSuite) Test_RegisterInvestmentTrade_Holdings() {
	type step struct {
		units int64
		day   int
		// warning is what the trade has to report, if anything.
		warning *negativeAt
	}

	tests := map[string]struct {
		steps []step
		held  int64
	}{
		"sale within the holdings":           {steps: []step{{10, 1, nil}, {-4, 2, nil}}, held: 6},
		"sale of everything":                 {steps: []step{{10, 1, nil}, {-10, 2, nil}}, held: 0},
		"sale of more than is held":          {steps: []step{{10, 1, nil}, {-50, 2, &negativeAt{-40, 2}}}, held: -40},
		"sale before any purchase":           {steps: []step{{-1, 1, &negativeAt{-1, 1}}}, held: -1},
		"backdated sale within the holdings": {steps: []step{{10, 1, nil}, {-4, 3, nil}, {-6, 2, nil}}, held: 0},
		"backdated sale that empties the position before a later sale": {
			steps: []step{{10, 1, nil}, {-4, 3, nil}, {-8, 2, &negativeAt{-2, 3}}},
			held:  -2,
		},
		"purchase and sale at the same instant":       {steps: []step{{5, 1, nil}, {-5, 1, nil}}, held: 0},
		"second sale at an instant that is exhausted": {steps: []step{{5, 1, nil}, {-5, 1, nil}, {-1, 1, &negativeAt{-1, 1}}}, held: -1},
		"sale recorded before the purchase it needs": {
			steps: []step{{-10, 2, &negativeAt{-10, 2}}, {10, 1, nil}},
			held:  0,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			entriesBefore := s.Count[model.JournalEntry]()

			for i, st := range tc.steps {
				activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, s.trade(fmt.Sprintf("%s-%d", name, i), asset, st.units, st.day))
				require.NoError(s.T(), err, "step %d", i)
				var result lt.RegisterInvestmentTradeResult
				require.NoError(s.T(), activityResult.Get(&result))

				if st.warning == nil {
					s.requireWarnings(result.Warnings, asset)
					continue
				}
				s.requireWarnings(result.Warnings, asset, expectedWarning{s.financialAccount, *st.warning})
			}

			require.Equal(s.T(), entriesBefore+len(tc.steps), s.Count[model.JournalEntry](), "a trade is written even when the holding is negative")
			require.True(s.T(), s.unitsHeld(s.financialAccount, asset).Equal(decimal.NewFromInt(tc.held)))
		})
	}
}

// Test_RegisterInvestmentTrade_Idempotent checks that a retry returns the first
// call's rows and the warnings as they are now: the sale warns when it is made,
// and no longer once the purchase it needed has been backfilled.
func (s *UnitTestSuite) Test_RegisterInvestmentTrade_Idempotent() {
	sale := s.trade("idempotent-sale", s.asset, -10, 2)

	first, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, sale)
	require.NoError(s.T(), err)
	var created lt.RegisterInvestmentTradeResult
	require.NoError(s.T(), first.Get(&created))
	s.requireWarnings(created.Warnings, s.asset, expectedWarning{s.financialAccount, negativeAt{-10, 2}})

	again, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, sale)
	require.NoError(s.T(), err)
	var retried lt.RegisterInvestmentTradeResult
	require.NoError(s.T(), again.Get(&retried))
	require.Equal(s.T(), created.JournalEntryID, retried.JournalEntryID)
	require.Equal(s.T(), created.PositionMovementID, retried.PositionMovementID)
	require.Equal(s.T(), created.Warnings, retried.Warnings)
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())

	s.seedUnits(s.financialAccount, s.asset, 10, 1)

	later, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, sale)
	require.NoError(s.T(), err)
	var afterBackfill lt.RegisterInvestmentTradeResult
	require.NoError(s.T(), later.Get(&afterBackfill))
	require.Equal(s.T(), created.JournalEntryID, afterBackfill.JournalEntryID)
	require.Empty(s.T(), afterBackfill.Warnings)
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())
}

func (s *UnitTestSuite) Test_RegisterInvestmentTrade_ClosedFinancialAccount() {
	first := s.trade("before-close", s.asset, 10, 1)
	_, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, first)
	require.NoError(s.T(), err)

	closedAt := time.Now().UTC()
	s.financialAccount.ClosedAt = &closedAt
	_, err = s.DB.NewUpdate().Model(s.financialAccount).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, s.trade("after-close", s.asset, 5, 2))
	require.ErrorContains(s.T(), err, "is closed")
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())

	// A retry of a trade made while the account was open still returns it.
	_, err = s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, first)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())
}

// Test_RegisterInvestmentTrade_WaitsForConcurrentWriter checks that a trade
// waits for a transaction that has locked the financial account, and then
// counts what it wrote. A concurrent sale of 4 out of 6 units is held open
// here, so without the lock the second sale would see 6 units and report no
// warning, although the position is left at -2.
func (s *UnitTestSuite) Test_RegisterInvestmentTrade_WaitsForConcurrentWriter() {
	_, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, s.trade("stock-up", s.asset, 6, 1))
	require.NoError(s.T(), err)

	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	defer func() { _ = writer.Rollback() }()
	_, err = writer.ExecContext(s.T().Context(), "SELECT id FROM account.financial_account WHERE id = ? FOR NO KEY UPDATE", s.financialAccount.ID)
	require.NoError(s.T(), err)
	_, err = writer.NewInsert().Model(temporaltesting.WithMovementDefaults(&model.PositionMovement{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            s.asset.ID,
		UnitsDelta:         decimal.NewFromInt(-4),
		OccurredAt:         holdingsBaseDay.AddDate(0, 0, 2),
	})).Exec(s.T().Context())
	require.NoError(s.T(), err)

	sold := make(chan *lt.RegisterInvestmentTradeResult, 1)
	go func() {
		result, err := s.pa.RegisterInvestmentTrade(s.T().Context(), s.trade("racing-sale", s.asset, -4, 3))
		require.NoError(s.T(), err)
		sold <- result
	}()

	select {
	case <-sold:
		s.T().Fatal("the sale finished without waiting for the transaction holding the account")
	case <-time.After(500 * time.Millisecond):
	}

	require.NoError(s.T(), writer.Commit())
	result := <-sold
	s.requireWarnings(result.Warnings, s.asset, expectedWarning{s.financialAccount, negativeAt{-2, 3}})
}
