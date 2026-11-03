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
			IdempotencyKey: "3f2a", BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
			Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*RecordExpenseInput)
		wantError string
	}{
		"an expense":              {mutate: func(*RecordExpenseInput) {}},
		"an expense with a note":  {mutate: func(i *RecordExpenseInput) { i.Description = "Esselunga" }},
		"a blank idempotency key": {mutate: func(i *RecordExpenseInput) { i.IdempotencyKey = " " }, wantError: "validation error"},
		"no balance":              {mutate: func(i *RecordExpenseInput) { i.BalanceLedgerAccountID = 0 }, wantError: "validation error"},
		"no category":             {mutate: func(i *RecordExpenseInput) { i.CategoryLedgerAccountID = 0 }, wantError: "validation error"},
		"a zero amount":           {mutate: func(i *RecordExpenseInput) { i.Amount = decimal.RequireFromString("0.00") }, wantError: "validation error"},
		"a negative amount":       {mutate: func(i *RecordExpenseInput) { i.Amount = decimal.RequireFromString("-12.50") }, wantError: "validation error"},
		"no occurrence date":      {mutate: func(i *RecordExpenseInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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
