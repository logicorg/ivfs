package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUpdateExpenseInput_Validate(t *testing.T) {
	valid := func() UpdateExpenseInput {
		return UpdateExpenseInput{
			JournalEntryID: 41, BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
			Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*UpdateExpenseInput)
		wantError string
	}{
		"an expense":         {mutate: func(*UpdateExpenseInput) {}},
		"with a note":        {mutate: func(i *UpdateExpenseInput) { i.Description = "a note" }},
		"no journal entry":   {mutate: func(i *UpdateExpenseInput) { i.JournalEntryID = 0 }, wantError: "validation error"},
		"a negative entry":   {mutate: func(i *UpdateExpenseInput) { i.JournalEntryID = -1 }, wantError: "validation error"},
		"no balance":         {mutate: func(i *UpdateExpenseInput) { i.BalanceLedgerAccountID = 0 }, wantError: "validation error"},
		"no category":        {mutate: func(i *UpdateExpenseInput) { i.CategoryLedgerAccountID = 0 }, wantError: "validation error"},
		"a zero amount":      {mutate: func(i *UpdateExpenseInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":  {mutate: func(i *UpdateExpenseInput) { i.Amount = decimal.NewFromInt(-12) }, wantError: "validation error"},
		"no occurrence date": {mutate: func(i *UpdateExpenseInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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

// The expense an update states is the one that would be recorded, with no key.
func TestUpdateExpenseInput_Expense(t *testing.T) {
	input := UpdateExpenseInput{
		JournalEntryID: 41, BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
		Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		Description: "a note",
	}

	require.Equal(t, RecordExpenseInput{
		BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
		Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), Description: "a note",
	}, input.Expense())
}
