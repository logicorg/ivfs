package investments

import (
	"testing"
	"time"

	it "github.com/logicorg/ivfs/finance/common/types/investmenttypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// TestBuildVest checks that a vest credits the income with the value of every
// unit vested, debits "Investments" and the tax category with the values of
// the units delivered and withheld, each rounded to the currency's precision,
// and adds the delivered units at their value.
func TestBuildVest(t *testing.T) {
	occurredAt := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	b := &brokerage{currency: euro, cash: cash.ID, investments: investments.ID}
	categories := &vestCategories{income: compensation.ID, tax: taxWithheld.ID}
	vest := func(vested, withheld, value string) it.VestInput {
		in := it.VestInput{
			IdempotencyKey: "a71f", FinancialAccountID: ibkr, AssetID: 7,
			UnitsVested: decimal.RequireFromString(vested), FairMarketValue: decimal.RequireFromString(value),
			CurrencyCode: "EUR", IncomeCategoryLedgerAccountID: compensation.ID, OccurredAt: occurredAt,
		}
		if withheld != "" {
			in.UnitsWithheld, in.TaxCategoryLedgerAccountID = decimal.RequireFromString(withheld), taxWithheld.ID
		}
		return in
	}
	postings := func(delivered, withheld, income string) []lt.PostingInput {
		result := []lt.PostingInput{{LedgerAccountID: investments.ID, Amount: decimal.RequireFromString(delivered), Side: model.PostingDebit}}
		if withheld != "" {
			result = append(result, lt.PostingInput{LedgerAccountID: taxWithheld.ID, Amount: decimal.RequireFromString(withheld), Side: model.PostingDebit})
		}
		return append(result, lt.PostingInput{LedgerAccountID: compensation.ID, Amount: decimal.RequireFromString(income), Side: model.PostingCredit})
	}

	tests := map[string]struct {
		input       it.VestInput
		postings    []lt.PostingInput
		delivered   string
		description *string
	}{
		"with units withheld for tax and a note": {
			input: func() it.VestInput {
				in := vest("100", "40", "10.00")
				in.Description = "Q3 vest"
				return in
			}(),
			postings:    postings("600", "400", "1000"),
			delivered:   "60",
			description: ptr.NewPtr("Q3 vest"),
		},
		"without withholding": {
			input:     vest("12.5", "", "10.00"),
			postings:  postings("125", "", "125"),
			delivered: "12.5",
		},
		// Each half cent rounds up, so the income, their sum, is a cent more
		// than the value of the units vested.
		"with values rounded to the currency's precision": {
			input:     vest("2", "1", "0.125"),
			postings:  postings("0.13", "0.13", "0.26"),
			delivered: "1",
		},
		"with a withholding worth less than a cent": {
			input:     vest("100", "0.001", "1"),
			postings:  postings("100", "", "100"),
			delivered: "99.999",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			want := lt.RegisterVestInput{
				Entry: lt.PostJournalEntryInput{
					IdempotencyKey: "vest:a71f",
					OccurredAt:     occurredAt,
					Description:    tc.description,
					Postings:       tc.postings,
				},
				FinancialAccountID:  ibkr,
				AssetID:             7,
				Units:               decimal.RequireFromString(tc.delivered),
				CostBasisAmount:     tc.postings[0].Amount,
				CostBasisCurrencyID: euro.ID,
			}
			require.Equal(t, normalizeVest(want), normalizeVest(buildVest(&tc.input, b, categories)))
		})
	}
}
