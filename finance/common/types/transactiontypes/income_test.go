package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRecordIncomeInput_Validate(t *testing.T) {
	valid := func() RecordIncomeInput {
		return RecordIncomeInput{
			IdempotencyKey: "77ab", BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
			Amount: decimal.NewFromInt(2000), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*RecordIncomeInput)
		wantError string
	}{
		"an income":               {mutate: func(*RecordIncomeInput) {}},
		"an income with a note":   {mutate: func(i *RecordIncomeInput) { i.Description = "October salary" }},
		"a blank idempotency key": {mutate: func(i *RecordIncomeInput) { i.IdempotencyKey = "" }, wantError: "validation error"},
		"no balance":              {mutate: func(i *RecordIncomeInput) { i.BalanceLedgerAccountID = 0 }, wantError: "validation error"},
		"no category":             {mutate: func(i *RecordIncomeInput) { i.CategoryLedgerAccountID = -1 }, wantError: "validation error"},
		"a zero amount":           {mutate: func(i *RecordIncomeInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":       {mutate: func(i *RecordIncomeInput) { i.Amount = decimal.NewFromInt(-2000) }, wantError: "validation error"},
		"no occurrence date":      {mutate: func(i *RecordIncomeInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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
