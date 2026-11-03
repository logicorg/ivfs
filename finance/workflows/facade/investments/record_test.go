package investments

import (
	"errors"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/testsuite"
)

// TestRecordInvestment checks that each kind finds the balances of its
// currency, whatever the case it is typed in, is registered as its builder
// makes it with the key prefixed by its kind, and passes the warnings on.
func (s *WorkflowUnitTestSuite) TestRecordInvestment() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input    it.RecordInvestmentInput
		want     lt.RegisterInvestmentTradeInput
		warnings []lt.NegativeHolding
	}{
		"a buy, in a currency code typed in lower case": {
			input: it.RecordInvestmentInput{Kind: it.KindBuy, Buy: &it.BuyAssetInput{
				IdempotencyKey: "5d0c", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(10),
				Amount: decimal.RequireFromString("1050.00"), Fee: decimal.RequireFromString("3.00"), CurrencyCode: " eur",
				OccurredAt: occurredAt, Description: "monthly plan",
			}},
			want: lt.RegisterInvestmentTradeInput{
				Entry: lt.PostJournalEntryInput{
					IdempotencyKey: "buy:5d0c",
					OccurredAt:     occurredAt,
					Description:    ptr.NewPtr("monthly plan"),
					Postings: []lt.PostingInput{
						{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(1053), Side: model.PostingDebit},
						{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1053), Side: model.PostingCredit},
					},
				},
				FinancialAccountID:  ibkr,
				AssetID:             7,
				UnitsDelta:          decimal.NewFromInt(10),
				CostBasisAmount:     ptr.NewPtr(decimal.NewFromInt(1053)),
				CostBasisCurrencyID: ptr.NewPtr(euro.ID),
			},
		},
		"a sale of more units than are held": {
			input: it.RecordInvestmentInput{Kind: it.KindSell, Sell: &it.SellAssetInput{
				IdempotencyKey: "8e41", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(100),
				Amount: decimal.NewFromInt(12000), CurrencyCode: "EUR", OccurredAt: occurredAt,
			}},
			want: lt.RegisterInvestmentTradeInput{
				Entry: lt.PostJournalEntryInput{
					IdempotencyKey: "sell:8e41",
					OccurredAt:     occurredAt,
					Postings: []lt.PostingInput{
						{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(12000), Side: model.PostingDebit},
						{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(12000), Side: model.PostingCredit},
					},
				},
				FinancialAccountID: ibkr,
				AssetID:            7,
				UnitsDelta:         decimal.NewFromInt(-100),
			},
			warnings: []lt.NegativeHolding{{FinancialAccountID: ibkr, AssetID: 7, At: occurredAt, Units: decimal.NewFromInt(-90)}},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			mockBrokerage(env)
			env.OnWorkflow(portfoliowf.RegisterInvestmentTrade, mock.Anything, tradeInput(tc.want)).
				Return(&lt.RegisterInvestmentTradeResult{JournalEntryID: 41, PositionMovementID: 42, Warnings: tc.warnings}, nil).Once()

			env.ExecuteWorkflow(RecordInvestment, tc.input)

			require.NoError(s.T(), env.GetWorkflowError())
			var got *it.TradeResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), int64(41), got.JournalEntryID)
			require.Equal(s.T(), int64(42), got.PositionMovementID)
			require.Len(s.T(), got.Warnings, len(tc.warnings))
			for i, warning := range tc.warnings {
				require.Equal(s.T(), warning.AssetID, got.Warnings[i].AssetID)
				require.True(s.T(), warning.Units.Equal(got.Warnings[i].Units))
			}
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordInvestment_Vest checks that a vest retrieves its categories, the
// tax one only when units are withheld, and is registered as its builder makes
// it with the key prefixed by its kind.
func (s *WorkflowUnitTestSuite) TestRecordInvestment_Vest() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	tests := map[string]struct {
		input    it.VestInput
		postings []lt.PostingInput
		units    decimal.Decimal
	}{
		"with units withheld for tax": {
			input: it.VestInput{
				IdempotencyKey: "a71f", FinancialAccountID: ibkr, AssetID: 7,
				UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.RequireFromString("10.00"),
				CurrencyCode: "eur", IncomeCategoryLedgerAccountID: compensation.ID, TaxCategoryLedgerAccountID: taxWithheld.ID,
				OccurredAt: occurredAt,
			},
			postings: []lt.PostingInput{
				{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(600), Side: model.PostingDebit},
				{LedgerAccountID: taxWithheld.ID, Amount: decimal.NewFromInt(400), Side: model.PostingDebit},
				{LedgerAccountID: compensation.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
			},
			units: decimal.NewFromInt(60),
		},
		// The unknown tax category would fail the vest if it were retrieved.
		"without withholding, leaving the tax category unused": {
			input: it.VestInput{
				IdempotencyKey: "a71f", FinancialAccountID: ibkr, AssetID: 7,
				UnitsVested: decimal.NewFromInt(100), FairMarketValue: decimal.RequireFromString("10.00"),
				CurrencyCode: "EUR", IncomeCategoryLedgerAccountID: compensation.ID, TaxCategoryLedgerAccountID: unknownCategory,
				OccurredAt: occurredAt,
			},
			postings: []lt.PostingInput{
				{LedgerAccountID: investments.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingDebit},
				{LedgerAccountID: compensation.ID, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
			},
			units: decimal.NewFromInt(100),
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			want := normalizeVest(lt.RegisterVestInput{
				Entry:               lt.PostJournalEntryInput{IdempotencyKey: "vest:a71f", OccurredAt: occurredAt, Postings: tc.postings},
				FinancialAccountID:  ibkr,
				AssetID:             7,
				Units:               tc.units,
				CostBasisAmount:     tc.postings[0].Amount,
				CostBasisCurrencyID: euro.ID,
			})
			env := s.NewTestWorkflowEnvironment()
			mockBrokerage(env)
			mockCategories(env)
			env.OnWorkflow(portfoliowf.RegisterVest, mock.Anything, mock.MatchedBy(func(got lt.RegisterVestInput) bool {
				return assert.ObjectsAreEqual(want, normalizeVest(got))
			})).Return(&lt.RegisterVestResult{JournalEntryID: 61, PositionMovementID: 62}, nil).Once()

			env.ExecuteWorkflow(RecordInvestment, it.RecordInvestmentInput{Kind: it.KindVest, Vest: &tc.input})

			require.NoError(s.T(), env.GetWorkflowError())
			var got *it.TradeResult
			require.NoError(s.T(), env.GetWorkflowResult(&got))
			require.Equal(s.T(), &it.TradeResult{JournalEntryID: 61, PositionMovementID: 62}, got)
			env.AssertExpectations(s.T())
		})
	}
}

// TestRecordInvestment_Errors checks that an invalid request, an unknown
// currency, an account without the two balances in it, or a vest's category of
// the wrong kind or currency fail before anything is written, since neither
// register workflow is mocked where a case does not reach it, and that a
// failing write's error is returned.
func (s *WorkflowUnitTestSuite) TestRecordInvestment_Errors() {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	buy := func(currencyCode string) it.RecordInvestmentInput {
		return it.RecordInvestmentInput{Kind: it.KindBuy, Buy: &it.BuyAssetInput{
			IdempotencyKey: "5d0c", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(10),
			Amount: decimal.NewFromInt(1050), CurrencyCode: currencyCode, OccurredAt: occurredAt,
		}}
	}
	sell := func(currencyCode string) it.RecordInvestmentInput {
		return it.RecordInvestmentInput{Kind: it.KindSell, Sell: &it.SellAssetInput{
			IdempotencyKey: "8e41", FinancialAccountID: ibkr, AssetID: 7, Units: decimal.NewFromInt(4),
			Amount: decimal.NewFromInt(480), CurrencyCode: currencyCode, OccurredAt: occurredAt,
		}}
	}
	vest := func(income, tax int64) it.RecordInvestmentInput {
		return it.RecordInvestmentInput{Kind: it.KindVest, Vest: &it.VestInput{
			IdempotencyKey: "a71f", FinancialAccountID: ibkr, AssetID: 7,
			UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.NewFromInt(10),
			CurrencyCode: "EUR", IncomeCategoryLedgerAccountID: income, TaxCategoryLedgerAccountID: tax, OccurredAt: occurredAt,
		}}
	}
	failedTrade := func(err error) func(env *testsuite.TestWorkflowEnvironment) {
		return func(env *testsuite.TestWorkflowEnvironment) {
			env.OnWorkflow(portfoliowf.RegisterInvestmentTrade, mock.Anything, mock.Anything).Return(nil, err)
		}
	}

	tests := map[string]struct {
		input     it.RecordInvestmentInput
		mock      func(env *testsuite.TestWorkflowEnvironment)
		wantError string
	}{
		"a kind with another kind's facts": {
			input:     it.RecordInvestmentInput{Kind: it.KindSell, Buy: buy("EUR").Buy},
			wantError: `invalid parameter: an investment of kind "sell" needs its facts in the field "sell"`,
		},
		"a buy with invalid facts": {
			input:     it.RecordInvestmentInput{Kind: it.KindBuy, Buy: &it.BuyAssetInput{FinancialAccountID: ibkr, AssetID: 7, CurrencyCode: "EUR"}},
			wantError: "validation error",
		},
		"a buy in an unknown currency": {
			input:     buy("XYZ"),
			wantError: "retrieve currency XYZ",
		},
		"a sale in a currency the account holds no investments in": {
			input:     sell("USD"),
			wantError: `invalid parameter: financial account 20 has no "Cash" and "Investments" balances in USD: open it again as a brokerage with USD to add them`,
		},
		"a failed buy": {
			input:     buy("EUR"),
			mock:      failedTrade(errors.New("resource FinancialAccount with ID: 20 is closed")),
			wantError: "register buy: ",
		},
		"a vest whose income category is in another currency": {
			input:     vest(compensationUSD.ID, taxWithheld.ID),
			wantError: `invalid parameter: "RSU compensation (USD)" is not in EUR, the currency of the vest: pick a category in EUR`,
		},
		"a vest whose income category is an expense one": {
			input:     vest(taxWithheld.ID, taxWithheld.ID),
			wantError: `invalid parameter: "Tax withheld" is an expense category, not an income one`,
		},
		"a vest whose tax category is a balance": {
			input:     vest(compensation.ID, cash.ID),
			wantError: `invalid parameter: "Cash" is not an expense category: its kind is asset`,
		},
		"a vest whose tax category is unknown": {
			input:     vest(compensation.ID, unknownCategory),
			wantError: "retrieve ledger account 999: ",
		},
		"a failed vest": {
			input: vest(compensation.ID, taxWithheld.ID),
			mock: func(env *testsuite.TestWorkflowEnvironment) {
				env.OnWorkflow(portfoliowf.RegisterVest, mock.Anything, mock.Anything).
					Return(nil, errors.New("resource LedgerAccount is closed"))
			},
			wantError: "register vest: ",
		},
		"a failed sale": {
			input:     sell("EUR"),
			mock:      failedTrade(errors.New("resource Asset not found with ID: 7")),
			wantError: "resource Asset not found with ID: 7",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			env := s.NewTestWorkflowEnvironment()
			if tc.mock != nil {
				tc.mock(env)
			}
			mockBrokerage(env)
			mockCategories(env)

			env.ExecuteWorkflow(RecordInvestment, tc.input)

			require.ErrorContains(s.T(), env.GetWorkflowError(), tc.wantError)
		})
	}
}
