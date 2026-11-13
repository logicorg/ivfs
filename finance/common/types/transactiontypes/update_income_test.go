package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUpdateIncomeInput_Validate(t *testing.T) {
	valid := func() UpdateIncomeInput {
		return UpdateIncomeInput{
			JournalEntryID: 41, BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
			Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*UpdateIncomeInput)
		wantError string
	}{
		"an income":          {mutate: func(*UpdateIncomeInput) {}},
		"with a note":        {mutate: func(i *UpdateIncomeInput) { i.Description = "a note" }},
		"no journal entry":   {mutate: func(i *UpdateIncomeInput) { i.JournalEntryID = 0 }, wantError: "validation error"},
		"a negative entry":   {mutate: func(i *UpdateIncomeInput) { i.JournalEntryID = -1 }, wantError: "validation error"},
		"no balance":         {mutate: func(i *UpdateIncomeInput) { i.BalanceLedgerAccountID = 0 }, wantError: "validation error"},
		"no category":        {mutate: func(i *UpdateIncomeInput) { i.CategoryLedgerAccountID = 0 }, wantError: "validation error"},
		"a zero amount":      {mutate: func(i *UpdateIncomeInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":  {mutate: func(i *UpdateIncomeInput) { i.Amount = decimal.NewFromInt(-12) }, wantError: "validation error"},
		"no occurrence date": {mutate: func(i *UpdateIncomeInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
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

// The income an update states is the one that would be recorded, with no key.
func TestUpdateIncomeInput_Income(t *testing.T) {
	input := UpdateIncomeInput{
		JournalEntryID: 41, BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
		Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		Description: "a note",
	}

	require.Equal(t, RecordIncomeInput{
		BalanceLedgerAccountID: 1, CategoryLedgerAccountID: 2,
		Amount: decimal.RequireFromString("12.50"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), Description: "a note",
	}, input.Income())
}
