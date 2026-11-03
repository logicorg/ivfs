package investmenttypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRecordInvestmentInput_Validate(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	buy := func() *BuyAssetInput {
		return &BuyAssetInput{
			IdempotencyKey: "5d0c", FinancialAccountID: 1, AssetID: 2,
			Units: decimal.NewFromInt(10), Amount: decimal.RequireFromString("1050.00"), CurrencyCode: "EUR",
			OccurredAt: occurredAt,
		}
	}
	sell := func() *SellAssetInput {
		return &SellAssetInput{
			IdempotencyKey: "8e41", FinancialAccountID: 1, AssetID: 2,
			Units: decimal.NewFromInt(4), Amount: decimal.RequireFromString("480.00"), CurrencyCode: "EUR",
			OccurredAt: occurredAt,
		}
	}

	vest := func() *VestInput {
		return &VestInput{
			IdempotencyKey: "a71f", FinancialAccountID: 1, AssetID: 2,
			UnitsVested: decimal.NewFromInt(100), FairMarketValue: decimal.RequireFromString("10.00"), CurrencyCode: "USD",
			IncomeCategoryLedgerAccountID: 3, OccurredAt: occurredAt,
		}
	}

	tests := map[string]struct {
		input     RecordInvestmentInput
		wantError string
	}{
		"a buy":  {input: RecordInvestmentInput{Kind: KindBuy, Buy: buy()}},
		"a sale": {input: RecordInvestmentInput{Kind: KindSell, Sell: sell()}},
		"a vest": {input: RecordInvestmentInput{Kind: KindVest, Vest: vest()}},
		"no kind": {
			input:     RecordInvestmentInput{Buy: buy()},
			wantError: `invalid parameter: kind "" is not one of [buy sell vest]`,
		},
		"an unknown kind": {
			input:     RecordInvestmentInput{Kind: "gift", Buy: buy()},
			wantError: `invalid parameter: kind "gift" is not one of [buy sell vest]`,
		},
		"a kind without its facts": {
			input:     RecordInvestmentInput{Kind: KindBuy},
			wantError: `invalid parameter: an investment of kind "buy" needs its facts in the field "buy", and no other kind's`,
		},
		"a kind with another kind's facts": {
			input:     RecordInvestmentInput{Kind: KindSell, Buy: buy()},
			wantError: `invalid parameter: an investment of kind "sell" needs its facts in the field "sell", and no other kind's`,
		},
		"a kind with its facts and another kind's": {
			input:     RecordInvestmentInput{Kind: KindBuy, Buy: buy(), Sell: sell()},
			wantError: `invalid parameter: an investment of kind "buy" needs its facts in the field "buy", and no other kind's`,
		},
		"a vest with a sale's facts": {
			input:     RecordInvestmentInput{Kind: KindVest, Sell: sell()},
			wantError: `invalid parameter: an investment of kind "vest" needs its facts in the field "vest", and no other kind's`,
		},
		"a vest with invalid facts": {
			input: func() RecordInvestmentInput {
				v := vest()
				v.UnitsWithheld = v.UnitsVested
				return RecordInvestmentInput{Kind: KindVest, Vest: v}
			}(),
			wantError: "invalid parameter: the units withheld for tax must be fewer than the units vested",
		},
		"a buy with invalid facts": {
			input:     RecordInvestmentInput{Kind: KindBuy, Buy: &BuyAssetInput{FinancialAccountID: 1}},
			wantError: "validation error",
		},
		"a sale whose fee takes all the proceeds": {
			input: func() RecordInvestmentInput {
				s := sell()
				s.Fee = s.Amount
				return RecordInvestmentInput{Kind: KindSell, Sell: s}
			}(),
			wantError: "invalid parameter: the fee of a sale must be less than its amount",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.input.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
