package portfolio

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestRegisterInvestmentTradeValidation() {
	tests := []struct {
		name string
		in   lt.RegisterInvestmentTradeInput
	}{
		{name: "missing account"},
		{
			name: "missing asset",
			in:   lt.RegisterInvestmentTradeInput{FinancialAccountID: 1},
		},
		{
			name: "missing idempotency key",
			in: lt.RegisterInvestmentTradeInput{
				FinancialAccountID: 1,
				AssetID:            1,
				UnitsDelta:         decimal.NewFromInt(1),
			},
		},
	}
	for _, tt := range tests {
		s.Run(tt.name, func() {
			_, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, tt.in)
			require.Error(s.T(), err)
		})
	}
}

func (s *UnitTestSuite) TestRegisterInvestmentTradeAndPosition() {
	// test RegisterInvestmentTrade activity

	input := lt.RegisterInvestmentTradeInput{
		FinancialAccountID:  s.financialAccount.ID,
		AssetID:             s.asset.ID,
		UnitsDelta:          decimal.NewFromInt(1),
		CostBasisAmount:     ptr.NewPtr(decimal.NewFromInt(100)),
		CostBasisCurrencyID: ptr.Int64(s.eur.ID),
		Entry: lt.PostJournalEntryInput{
			IdempotencyKey: "buy-aapl-1", OccurredAt: time.Now().UTC(),
			Postings: []lt.PostingInput{
				{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(100), Side: model.PostingDebit},
				{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(100), Side: model.PostingCredit},
			},
		},
	}

	activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, input)
	require.NoError(s.T(), err)

	var result lt.RegisterInvestmentTradeResult
	require.NoError(s.T(), activityResult.Get(&result))
	require.NotZero(s.T(), result.JournalEntryID)
	require.NotZero(s.T(), result.PositionMovementID)

	// test PositionAt activtiy

	params := lt.PositionAtParams{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            s.asset.ID,
	}

	positionResult, err := s.Env.ExecuteActivity(s.pa.PositionAt, params)
	require.NoError(s.T(), err)

	var position lt.PositionAtResult
	require.NoError(s.T(), positionResult.Get(&position))
	require.True(s.T(), position.Units.Equal(decimal.NewFromInt(1)))

	// test RegisterInvestmentTrade idempotency

	retryResult, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, input)
	require.NoError(s.T(), err)

	var retry lt.RegisterInvestmentTradeResult
	require.NoError(s.T(), retryResult.Get(&retry))
	require.Equal(s.T(), result.JournalEntryID, retry.JournalEntryID)
	require.Equal(s.T(), result.PositionMovementID, retry.PositionMovementID)
	count, err := s.DB.NewSelect().Model((*model.PositionMovement)(nil)).Where("journal_entry_id = ?", result.JournalEntryID).Count(s.T().Context())
	require.NoError(s.T(), err)
	require.Equal(s.T(), 1, count)
}

// The cost basis is in a currency of its own, so a trade can carry it in another
// currency than the one it was paid in, and the amount and the currency are
// given together or not at all.
func (s *UnitTestSuite) Test_RegisterInvestmentTrade_CostBasis() {
	usd := testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, usd)

	tests := map[string]struct {
		amount   *decimal.Decimal
		currency *int64
		expErr   string
	}{
		"in another currency than the cash": {amount: ptr.NewPtr(decimal.NewFromInt(110)), currency: ptr.Int64(usd.ID)},
		"unknown":                           {},
		"an amount without a currency":      {amount: ptr.NewPtr(decimal.NewFromInt(110)), expErr: "given together"},
		"a currency without an amount":      {currency: ptr.Int64(usd.ID), expErr: "given together"},
		"in a currency that does not exist": {amount: ptr.NewPtr(decimal.NewFromInt(110)), currency: ptr.Int64(999999), expErr: "Currency"},
	}

	day := 0
	for name, tc := range tests {
		day++
		s.Run(name, func() {
			input := s.trade("cost-basis-"+name, s.asset, 1, day)
			input.CostBasisAmount, input.CostBasisCurrencyID = tc.amount, tc.currency

			activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterInvestmentTrade, input)

			if tc.expErr != "" {
				require.ErrorContains(s.T(), err, tc.expErr)
				return
			}
			require.NoError(s.T(), err)

			var result lt.RegisterInvestmentTradeResult
			require.NoError(s.T(), activityResult.Get(&result))
			var stored model.PositionMovement
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", result.PositionMovementID).Scan(s.T().Context()))
			require.Equal(s.T(), model.PositionMovementTrade, stored.Kind)
			require.Equal(s.T(), tc.currency, stored.CostBasisCurrencyID)
			if tc.amount == nil {
				require.Nil(s.T(), stored.CostBasisAmount)
				return
			}
			require.True(s.T(), tc.amount.Equal(*stored.CostBasisAmount))
		})
	}
}
