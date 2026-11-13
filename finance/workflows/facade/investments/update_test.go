package investments

import (
	"errors"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// replacement matches the replace want by what it records rather than by its
// representation, as tradeInput does: a decimal such as 1053.00 arrives as
// 1053, equal in value but not in its fields.
func replacement(want lt.ReplaceJournalEntryInput) any {
	sameDecimal := func(a, b *decimal.Decimal) bool { return (a == nil) == (b == nil) && (a == nil || a.Equal(*b)) }
	sameID := func(a, b *int64) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }
	sameString := func(a, b *string) bool { return (a == nil) == (b == nil) && (a == nil || *a == *b) }

	return mock.MatchedBy(func(got lt.ReplaceJournalEntryInput) bool {
		if got.JournalEntryID != want.JournalEntryID || !got.OccurredAt.Equal(want.OccurredAt) ||
			!sameString(got.Description, want.Description) || len(got.Metadata) != 0 ||
			len(got.Postings) != len(want.Postings) || len(got.Movements) != len(want.Movements) {
			return false
		}
		for i, posting := range want.Postings {
			if got.Postings[i].LedgerAccountID != posting.LedgerAccountID || got.Postings[i].Side != posting.Side ||
				!got.Postings[i].Amount.Equal(posting.Amount) {
				return false
			}
		}
		for i, movement := range want.Movements {
			m := got.Movements[i]
			if m.Kind != movement.Kind || m.FinancialAccountID != movement.FinancialAccountID || m.AssetID != movement.AssetID ||
				!m.UnitsDelta.Equal(movement.UnitsDelta) || !sameDecimal(m.CostBasisAmount, movement.CostBasisAmount) ||
				!sameID(m.CostBasisCurrencyID, movement.CostBasisCurrencyID) {
				return false
			}
		}
		return true
	})
}

// TestUpdateInvestment checks that each kind replaces the entry with the
// postings and the movement its builder makes for RecordInvestment, and passes
// the movement's id and the warnings on.
func (s *WorkflowUnitTestSuite) TestUpdateInvestment() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		input    it.UpdateInvestmentInput
		want     lt.ReplaceJournalEntryInput
		warnings []lt.NegativeHolding
	}{
		{
			name: "a buy, in a currency code typed in lower case",
			input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindBuy, Buy: &it.UpdateBuyInput{
				FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(10),
				Amount: decimal.RequireFromString("1050.00"), Fee: decimal.RequireFromString("3.00"), CurrencyCode: " eur",
				OccurredAt: occurredAt, Description: "monthly plan",
			}},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 41, OccurredAt: occurredAt, Description: ptr.NewPtr("monthly plan"),
				Postings: []lt.PostingInput{
					{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(1053), Side: model.PostingDebit},
					{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1053), Side: model.PostingCredit},
				},
				Movements: []lt.ReplaceMovementInput{{
					Kind: model.PositionMovementTrade, FinancialAccountID: ibkr, AssetID: 7, UnitsDelta: decimal.NewFromInt(10),
					CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(1053)), CostBasisCurrencyID: ptr.NewPtr(euro.ID),
				}},
			},
		},
		{
			name: "a sale of more units than are held",
			input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindSell, Sell: &it.UpdateSellInput{
				FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(100),
				Amount: decimal.NewFromInt(12000), Fee: decimal.NewFromInt(3), CurrencyCode: "EUR", OccurredAt: occurredAt,
			}},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 41, OccurredAt: occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(11997), Side: model.PostingDebit},
					{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(11997), Side: model.PostingCredit},
				},
				Movements: []lt.ReplaceMovementInput{{
					Kind: model.PositionMovementTrade, FinancialAccountID: ibkr, AssetID: 7, UnitsDelta: decimal.NewFromInt(-100),
				}},
			},
			warnings: []lt.NegativeHolding{{FinancialAccountID: ibkr, AssetID: 7, At: occurredAt, Units: decimal.NewFromInt(-90)}},
		},
		{
			name: "a vest with units withheld, lowered below what was sold since",
			input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &it.UpdateVestInput{
				FinancialAccountID: ibkr, AssetID: 7,
				UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.RequireFromString("10.00"),
				CurrencyCode: "eur", IncomeCategoryLedgerAccountID: compensation.ID, TaxCategoryLedgerAccountID: taxWithheld.ID,
				OccurredAt: occurredAt,
			}},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 41, OccurredAt: occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(600), Side: model.PostingDebit},
					{LedgerAccountID: taxWithheld.ID, Amount: decimal.NewFromInt(400), Side: model.PostingDebit},
					{LedgerAccountID: compensation.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
				},
				Movements: []lt.ReplaceMovementInput{{
					Kind: model.PositionMovementVest, FinancialAccountID: ibkr, AssetID: 7, UnitsDelta: decimal.NewFromInt(60),
					CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(600)), CostBasisCurrencyID: ptr.NewPtr(euro.ID),
				}},
			},
			warnings: []lt.NegativeHolding{{FinancialAccountID: ibkr, AssetID: 7, At: occurredAt, Units: decimal.NewFromInt(-20)}},
		},
		{
			// The unknown tax category would fail the vest if it were retrieved.
			name: "a vest without withholding, leaving the tax category unused",
			input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &it.UpdateVestInput{
				FinancialAccountID: ibkr, AssetID: 7,
				UnitsVested: decimal.NewFromInt(100), FairMarketValue: decimal.RequireFromString("10.00"),
				CurrencyCode: "EUR", IncomeCategoryLedgerAccountID: compensation.ID, TaxCategoryLedgerAccountID: unknownCategory,
				OccurredAt: occurredAt,
			}},
			want: lt.ReplaceJournalEntryInput{
				JournalEntryID: 41, OccurredAt: occurredAt,
				Postings: []lt.PostingInput{
					{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingDebit},
					{LedgerAccountID: compensation.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
				},
				Movements: []lt.ReplaceMovementInput{{
					Kind: model.PositionMovementVest, FinancialAccountID: ibkr, AssetID: 7, UnitsDelta: decimal.NewFromInt(100),
					CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(1000)), CostBasisCurrencyID: ptr.NewPtr(euro.ID),
				}},
			},
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockBrokerage(env)
			mockCategories(env)
			env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, replacement(tc.want)).
				Return(&lt.ReplaceJournalEntryResult{JournalEntryID: 41, PositionMovementIDs: []int64{43}, Warnings: tc.warnings}, nil).Once()

			env.ExecuteWorkflow(UpdateInvestment, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *it.TradeResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), int64(41), got.JournalEntryID)
			require.Equal(s.T(), int64(43), got.PositionMovementID)
			require.Len(s.T(), got.Warnings, len(tc.warnings))
			for i, warning := range tc.warnings {
				require.Equal(s.T(), warning.AssetID, got.Warnings[i].AssetID)
				require.True(s.T(), warning.Units.Equal(got.Warnings[i].Units))
			}
			env.AssertExpectations(s.T())
		})
	}
}

// TestUpdateInvestment_Errors checks that an invalid request, an unknown
// currency, an account without the two balances in it, or a vest's category of
// the wrong kind or currency fail before anything is replaced, since
// ReplaceJournalEntry is mocked only where a case reaches it, and that a
// failing replace's error is returned.
func (s *WorkflowUnitTestSuite) TestUpdateInvestment_Errors() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	buy := it.UpdateBuyInput{FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(10), Amount: decimal.NewFromInt(1050), CurrencyCode: "EUR", OccurredAt: occurredAt}
	buyXYZ, sellUSD := buy, it.UpdateSellInput{FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(4), Amount: decimal.NewFromInt(480), CurrencyCode: "USD", OccurredAt: occurredAt}
	buyXYZ.CurrencyCode = "XYZ"
	vest := it.UpdateVestInput{
		FinancialAccountID: ibkr, AssetID: 7,
		UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.NewFromInt(10),
		CurrencyCode: "EUR", IncomeCategoryLedgerAccountID: compensation.ID, TaxCategoryLedgerAccountID: taxWithheld.ID, OccurredAt: occurredAt,
	}
	vestIncomeUSD, vestIncomeExpense, vestTaxBalance, vestTaxUnknown := vest, vest, vest, vest
	vestIncomeUSD.IncomeCategoryLedgerAccountID = compensationUSD.ID
	vestIncomeExpense.IncomeCategoryLedgerAccountID = taxWithheld.ID
	vestTaxBalance.TaxCategoryLedgerAccountID = cash.ID
	vestTaxUnknown.TaxCategoryLedgerAccountID = unknownCategory

	tests := []struct {
		name  string
		input it.UpdateInvestmentInput
		// replaceErr is what ReplaceJournalEntry fails with, when reached.
		replaceErr string
		wantError  string
	}{
		{name: "no entry", input: it.UpdateInvestmentInput{Kind: it.KindBuy, Buy: &buy}, wantError: "invalid parameter: an update needs the id of the journal entry"},
		{name: "a kind with another kind's facts", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindSell, Buy: &buy}, wantError: `invalid parameter: an investment of kind "sell" needs its facts in the field "sell"`},
		{name: "a buy with invalid facts", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindBuy, Buy: &it.UpdateBuyInput{FinancialAccountID: ibkr, AssetID: 7, CurrencyCode: "EUR"}}, wantError: "validation error"},
		{name: "a buy in an unknown currency", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindBuy, Buy: &buyXYZ}, wantError: "retrieve currency XYZ"},
		{name: "a sale in a currency the account holds no investments in", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindSell, Sell: &sellUSD}, wantError: `invalid parameter: financial account 20 has no "Cash" and "Investments" balances in USD`},
		{name: "a vest whose income category is in another currency", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &vestIncomeUSD}, wantError: `invalid parameter: "RSU compensation (USD)" is not in EUR, the currency of the vest`},
		{name: "a vest whose income category is an expense one", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &vestIncomeExpense}, wantError: `invalid parameter: "Tax withheld" is an expense category, not an income one`},
		{name: "a vest whose tax category is a balance", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &vestTaxBalance}, wantError: `invalid parameter: "Cash" is not an expense category: its kind is asset`},
		{name: "a vest whose tax category is unknown", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &vestTaxUnknown}, wantError: "retrieve ledger account 999: "},
		{name: "an unknown entry", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindBuy, Buy: &buy}, replaceErr: "resource JournalEntry not found with ID: 41", wantError: "update buy: "},
		{name: "a vest onto a closed account", input: it.UpdateInvestmentInput{JournalEntryID: 41, Kind: it.KindVest, Vest: &vest}, replaceErr: "resource FinancialAccount with ID: 20 is closed", wantError: "resource FinancialAccount with ID: 20 is closed"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.replaceErr != "" {
				env.OnWorkflow(journalwf.ReplaceJournalEntry, mock.Anything, mock.Anything).Return(nil, errors.New(tc.replaceErr))
			}
			mockBrokerage(env)
			mockCategories(env)

			env.ExecuteWorkflow(UpdateInvestment, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
