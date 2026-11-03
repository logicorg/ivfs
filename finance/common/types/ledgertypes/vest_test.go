package ledgertypes

import (
	"testing"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRegisterVestInput_Validate(t *testing.T) {
	// valid is a vest of 60 units worth 600, with 400 more withheld in shares. The
	// postings only have to balance, so income is credited and the delivered
	// value and the tax are debited.
	valid := func() RegisterVestInput {
		return RegisterVestInput{
			FinancialAccountID:  1,
			AssetID:             1,
			Units:               decimal.NewFromInt(60),
			CostBasisAmount:     decimal.NewFromInt(600),
			CostBasisCurrencyID: 1,
			Entry: PostJournalEntryInput{
				IdempotencyKey: "acme-rsu-2026-03-01",
				OccurredAt:     time.Now(),
				Postings: []PostingInput{
					{LedgerAccountID: 10, Amount: decimal.NewFromInt(1000), Side: model.PostingCredit},
					{LedgerAccountID: 20, Amount: decimal.NewFromInt(600), Side: model.PostingDebit},
					{LedgerAccountID: 30, Amount: decimal.NewFromInt(400), Side: model.PostingDebit},
				},
			},
		}
	}

	const (
		accepted   = ""
		constraint = "cue"
		rule       = "rule"
	)

	tests := map[string]struct {
		mutate func(*RegisterVestInput)
		// failure is how the input is rejected: by a CUE constraint, which is a
		// ValidationError, or by a rule CUE cannot express, which is an
		// ErrInvalidParameter.
		failure string
	}{
		"a vest": {mutate: func(*RegisterVestInput) {}},
		"fractional units": {mutate: func(i *RegisterVestInput) {
			i.Units = decimal.RequireFromString("0.000001")
		}},
		"no financial account": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.FinancialAccountID = 0
		}},
		"no asset": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.AssetID = 0
		}},
		"no idempotency key": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.Entry.IdempotencyKey = ""
		}},
		"zero units": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.Units = decimal.Zero
		}},
		"zero units written with decimals": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.Units = decimal.RequireFromString("0.00")
		}},
		"no cost basis": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.CostBasisAmount = decimal.Zero
		}},
		"no cost basis currency": {failure: constraint, mutate: func(i *RegisterVestInput) {
			i.CostBasisCurrencyID = 0
		}},
		"negative units, which a sale would have": {failure: rule, mutate: func(i *RegisterVestInput) {
			i.Units = decimal.NewFromInt(-60)
		}},
		"a negative cost basis": {failure: rule, mutate: func(i *RegisterVestInput) {
			i.CostBasisAmount = decimal.NewFromInt(-600)
		}},
		"an entry with a single posting": {failure: rule, mutate: func(i *RegisterVestInput) {
			i.Entry.Postings = i.Entry.Postings[:1]
		}},
		"an entry without a date": {failure: rule, mutate: func(i *RegisterVestInput) {
			i.Entry.OccurredAt = time.Time{}
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)

			err := input.Validate()

			switch tc.failure {
			case accepted:
				require.NoError(t, err)
			case constraint:
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
			case rule:
				require.ErrorAs(t, err, &internalerrors.ErrInvalidParameter{})
			}
		})
	}
}
