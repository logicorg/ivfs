package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRecordTransferInput_Validate(t *testing.T) {
	valid := func() RecordTransferInput {
		return RecordTransferInput{
			IdempotencyKey: "9c1e", FromLedgerAccountID: 1, ToLedgerAccountID: 2,
			Amount: decimal.NewFromInt(300), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*RecordTransferInput)
		wantError string
	}{
		"a transfer":                 {mutate: func(*RecordTransferInput) {}},
		"a transfer with an arrival": {mutate: func(i *RecordTransferInput) { i.ToAmount = decimal.RequireFromString("324.15") }},
		"a transfer with a note":     {mutate: func(i *RecordTransferInput) { i.Description = "pay the card off" }},
		"with tags":                  {mutate: func(i *RecordTransferInput) { i.TagIDs = []int64{3, 4} }},
		"with charges": {
			mutate: func(i *RecordTransferInput) {
				i.Charges = []Charge{{CategoryID: 7, Amount: decimal.NewFromInt(100)}, {CategoryID: 8, Amount: decimal.RequireFromString("199.99")}}
			},
		},
		"a charge with no category": {
			mutate:    func(i *RecordTransferInput) { i.Charges = []Charge{{Amount: decimal.NewFromInt(100)}} },
			wantError: "invalid parameter: a charge names the expense category it is booked under",
		},
		"a charge of zero": {
			mutate:    func(i *RecordTransferInput) { i.Charges = []Charge{{CategoryID: 7}} },
			wantError: "invalid parameter: a charge is a positive part of the amount, not 0",
		},
		"a negative charge": {
			mutate:    func(i *RecordTransferInput) { i.Charges = []Charge{{CategoryID: 7, Amount: decimal.NewFromInt(-1)}} },
			wantError: "invalid parameter: a charge is a positive part of the amount, not -1",
		},
		"charges taking the whole amount": {
			mutate: func(i *RecordTransferInput) {
				i.Charges = []Charge{{CategoryID: 7, Amount: decimal.NewFromInt(100)}, {CategoryID: 8, Amount: decimal.NewFromInt(200)}}
			},
			wantError: "invalid parameter: the charges add up to 300 of the 300 that leaves, so nothing reaches the other balance; record it as an expense",
		},
		"a tag id of zero":               {mutate: func(i *RecordTransferInput) { i.TagIDs = []int64{3, 0} }, wantError: "tag id 0"},
		"a blank idempotency key":        {mutate: func(i *RecordTransferInput) { i.IdempotencyKey = "\t" }, wantError: "validation error"},
		"no source":                      {mutate: func(i *RecordTransferInput) { i.FromLedgerAccountID = 0 }, wantError: "validation error"},
		"no destination":                 {mutate: func(i *RecordTransferInput) { i.ToLedgerAccountID = 0 }, wantError: "validation error"},
		"a zero amount":                  {mutate: func(i *RecordTransferInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":              {mutate: func(i *RecordTransferInput) { i.Amount = decimal.NewFromInt(-300) }, wantError: "validation error"},
		"a negative amount that arrives": {mutate: func(i *RecordTransferInput) { i.ToAmount = decimal.NewFromInt(-300) }, wantError: "validation error"},
		"no occurrence date":             {mutate: func(i *RecordTransferInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
		"the same balance twice": {
			mutate:    func(i *RecordTransferInput) { i.ToLedgerAccountID = i.FromLedgerAccountID },
			wantError: "invalid parameter: a transfer moves money between two different balances",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)
			err := input.Validate()
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}
