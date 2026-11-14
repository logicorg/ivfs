package transactiontypes

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestUpdateTransferInput_Validate(t *testing.T) {
	valid := func() UpdateTransferInput {
		return UpdateTransferInput{
			JournalEntryID: 42, FromLedgerAccountID: 1, ToLedgerAccountID: 2,
			Amount: decimal.NewFromInt(300), ToAmount: decimal.RequireFromString("324.15"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		}
	}

	tests := map[string]struct {
		mutate    func(*UpdateTransferInput)
		wantError string
	}{
		"a transfer":         {mutate: func(*UpdateTransferInput) {}},
		"without an arrival": {mutate: func(i *UpdateTransferInput) { i.ToAmount = decimal.Zero }},
		"no journal entry":   {mutate: func(i *UpdateTransferInput) { i.JournalEntryID = 0 }, wantError: "validation error"},
		"no source":          {mutate: func(i *UpdateTransferInput) { i.FromLedgerAccountID = 0 }, wantError: "validation error"},
		"no destination":     {mutate: func(i *UpdateTransferInput) { i.ToLedgerAccountID = 0 }, wantError: "validation error"},
		"a zero amount":      {mutate: func(i *UpdateTransferInput) { i.Amount = decimal.Zero }, wantError: "validation error"},
		"a negative amount":  {mutate: func(i *UpdateTransferInput) { i.Amount = decimal.NewFromInt(-300) }, wantError: "validation error"},
		"a negative arrival": {mutate: func(i *UpdateTransferInput) { i.ToAmount = decimal.NewFromInt(-300) }, wantError: "validation error"},
		"no occurrence date": {mutate: func(i *UpdateTransferInput) { i.OccurredAt = time.Time{} }, wantError: "missing required parameters: OccurredAt"},
		"the same balance twice": {
			mutate:    func(i *UpdateTransferInput) { i.ToLedgerAccountID = i.FromLedgerAccountID },
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

// The transfer an update states is the one that would be recorded, with no key.
func TestUpdateTransferInput_Transfer(t *testing.T) {
	input := UpdateTransferInput{
		JournalEntryID: 42, FromLedgerAccountID: 1, ToLedgerAccountID: 2,
		Amount: decimal.NewFromInt(300), ToAmount: decimal.RequireFromString("324.15"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC),
		Description: "a note",
	}

	require.Equal(t, RecordTransferInput{
		FromLedgerAccountID: 1, ToLedgerAccountID: 2,
		Amount: decimal.NewFromInt(300), ToAmount: decimal.RequireFromString("324.15"), OccurredAt: time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC), Description: "a note",
	}, input.Transfer())
}
