package investmenttypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUpdateInvestmentInput_Validate(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	buy := UpdateBuyInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(10), Amount: decimal.RequireFromString("1050.00"), CurrencyCode: "EUR", OccurredAt: occurredAt}
	sell := UpdateSellInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(4), Amount: decimal.RequireFromString("480.00"), CurrencyCode: "EUR", OccurredAt: occurredAt}
	vest := UpdateVestInput{FinancialAccountID: 1, AssetID: 2, UnitsVested: decimal.NewFromInt(100), FairMarketValue: decimal.RequireFromString("10.00"), CurrencyCode: "USD", IncomeCategoryLedgerAccountID: 3, OccurredAt: occurredAt}

	buyNoDate, sellFeeAll, vestAllWithheld, vestNoTaxCategory := buy, sell, vest, vest
	buyNoDate.OccurredAt = time.Time{}
	sellFeeAll.Fee = sell.Amount
	vestAllWithheld.UnitsWithheld = vest.UnitsVested
	vestNoTaxCategory.UnitsWithheld = decimal.NewFromInt(40)

	tests := []struct {
		name      string
		input     UpdateInvestmentInput
		wantError string
	}{
		{name: "a buy", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindBuy, Buy: &buy}},
		{name: "a sale", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindSell, Sell: &sell}},
		{name: "a vest", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindVest, Vest: &vest}},
		{name: "no entry", input: UpdateInvestmentInput{Kind: KindBuy, Buy: &buy}, wantError: "invalid parameter: an update needs the id of the journal entry to correct"},
		{name: "a negative entry", input: UpdateInvestmentInput{JournalEntryID: -1, Kind: KindBuy, Buy: &buy}, wantError: "invalid parameter: an update needs the id of the journal entry to correct"},
		{name: "no kind", input: UpdateInvestmentInput{JournalEntryID: 41, Buy: &buy}, wantError: `invalid parameter: kind "" is not one of [buy sell vest]`},
		{name: "an unknown kind", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: "gift", Buy: &buy}, wantError: `invalid parameter: kind "gift" is not one of [buy sell vest]`},
		{name: "a kind without its facts", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindBuy}, wantError: `invalid parameter: an investment of kind "buy" needs its facts in the field "buy", and no other kind's`},
		{name: "a kind with another kind's facts", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindSell, Buy: &buy}, wantError: `invalid parameter: an investment of kind "sell" needs its facts in the field "sell", and no other kind's`},
		{name: "a kind with its facts and another kind's", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindBuy, Buy: &buy, Sell: &sell}, wantError: `invalid parameter: an investment of kind "buy" needs its facts in the field "buy", and no other kind's`},
		{name: "a buy with invalid facts", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindBuy, Buy: &UpdateBuyInput{FinancialAccountID: 1}}, wantError: "validation error"},
		{name: "a sale with invalid facts", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindSell, Sell: &UpdateSellInput{FinancialAccountID: 1}}, wantError: "validation error"},
		{name: "a vest with invalid facts", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindVest, Vest: &UpdateVestInput{FinancialAccountID: 1}}, wantError: "validation error"},
		{name: "a buy without a date", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindBuy, Buy: &buyNoDate}, wantError: "missing required parameters: OccurredAt"},
		{name: "a sale whose fee takes all the proceeds", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindSell, Sell: &sellFeeAll}, wantError: "invalid parameter: the fee of a sale must be less than its amount"},
		{name: "a vest that delivers nothing", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindVest, Vest: &vestAllWithheld}, wantError: "invalid parameter: the units withheld for tax must be fewer than the units vested"},
		{name: "a vest with withholding and no tax category", input: UpdateInvestmentInput{JournalEntryID: 41, Kind: KindVest, Vest: &vestNoTaxCategory}, wantError: "missing required parameters: TaxCategoryLedgerAccountID"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.input.Validate()

			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

// The facts an update states are the ones that would be recorded, with no key.
func TestUpdateInvestmentInput_Facts(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)

	buy := UpdateBuyInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(10), Amount: decimal.NewFromInt(1050), Fee: decimal.NewFromInt(3), CurrencyCode: "EUR", OccurredAt: occurredAt, Description: "a note"}
	require.Equal(t, &BuyAssetInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(10), Amount: decimal.NewFromInt(1050), Fee: decimal.NewFromInt(3), CurrencyCode: "EUR", OccurredAt: occurredAt, Description: "a note"}, buy.Buy())

	sell := UpdateSellInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(4), Amount: decimal.NewFromInt(480), Fee: decimal.NewFromInt(3), CurrencyCode: "EUR", OccurredAt: occurredAt, Description: "a note"}
	require.Equal(t, &SellAssetInput{FinancialAccountID: 1, AssetID: 2, Units: decimal.NewFromInt(4), Amount: decimal.NewFromInt(480), Fee: decimal.NewFromInt(3), CurrencyCode: "EUR", OccurredAt: occurredAt, Description: "a note"}, sell.Sell())

	vest := UpdateVestInput{FinancialAccountID: 1, AssetID: 2, UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.NewFromInt(10), CurrencyCode: "USD", IncomeCategoryLedgerAccountID: 3, TaxCategoryLedgerAccountID: 4, OccurredAt: occurredAt, Description: "a note"}
	require.Equal(t, &VestInput{FinancialAccountID: 1, AssetID: 2, UnitsVested: decimal.NewFromInt(100), UnitsWithheld: decimal.NewFromInt(40), FairMarketValue: decimal.NewFromInt(10), CurrencyCode: "USD", IncomeCategoryLedgerAccountID: 3, TaxCategoryLedgerAccountID: 4, OccurredAt: occurredAt, Description: "a note"}, vest.Vest())
}
