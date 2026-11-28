package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRecordExpenseInput_Validate(t *testing.T) {
	valid := func() RecordExpenseInput {
		return RecordExpenseInput{
			IdempotencyKey: "3f2a", BalanceLedgerAccountID: 1, CategoryID: 2,
			Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*RecordExpenseInput)
		wantError string
	}{
		"an expense":              {mutate: func(*RecordExpenseInput) {}},
		"an expense with a note":  {mutate: func(i *RecordExpenseInput) { i.Description = "Esselunga" }},
		"with tags":               {mutate: func(i *RecordExpenseInput) { i.TagIDs = []int64{3, 4} }},
		"a tag id of zero":        {mutate: func(i *RecordExpenseInput) { i.TagIDs = []int64{3, 0} }, wantError: "tag id 0"},
		"a blank idempotency key": {mutate: func(i *RecordExpenseInput) { i.IdempotencyKey = " " }, wantError: "validation error"},
		"no balance":              {mutate: func(i *RecordExpenseInput) { i.BalanceLedgerAccountID = 0 }, wantError: "validation error"},
		"no category":             {mutate: func(i *RecordExpenseInput) { i.CategoryID = 0 }, wantError: "names the categoryId it was spent on, or its lines"},
		"a negative category":     {mutate: func(i *RecordExpenseInput) { i.CategoryID = -1 }, wantError: "validation error"},
		"with lines": {mutate: func(i *RecordExpenseInput) {
			i.CategoryID, i.Lines = 0, []Split{{CategoryID: 2, Amount: decimal.NewFromInt(10)}, {CategoryID: 3, Amount: decimal.RequireFromString("2.5")}}
		}},
		"lines and a category": {
			mutate: func(i *RecordExpenseInput) {
				i.Lines = []Split{{CategoryID: 2, Amount: decimal.RequireFromString("12.50")}}
			},
			wantError: "omit categoryId",
		},
		"lines that do not add up": {
			mutate: func(i *RecordExpenseInput) {
				i.CategoryID, i.Lines = 0, []Split{{CategoryID: 2, Amount: decimal.NewFromInt(10)}, {CategoryID: 3, Amount: decimal.NewFromInt(2)}}
			},
			wantError: "the lines add up to 12, not the 12.5 paid",
		},
		"a line with no category": {
			mutate: func(i *RecordExpenseInput) {
				i.CategoryID, i.Lines = 0, []Split{{Amount: decimal.RequireFromString("12.50")}}
			},
			wantError: "a line names the expense category",
		},
		"a line of zero": {
			mutate: func(i *RecordExpenseInput) {
				i.CategoryID, i.Lines = 0, []Split{{CategoryID: 2, Amount: decimal.RequireFromString("12.50")}, {CategoryID: 3}}
			},
			wantError: "a line is a positive part of the amount, not 0",
		},
		"a zero amount":      {mutate: func(i *RecordExpenseInput) { i.Amount = decimal.RequireFromString("0.00") }, wantError: "validation error"},
		"a negative amount":  {mutate: func(i *RecordExpenseInput) { i.Amount = decimal.RequireFromString("-12.50") }, wantError: "validation error"},
		"no occurrence date": {mutate: func(i *RecordExpenseInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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
