package ledgertypes

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestReplaceJournalEntryInput_Validate(t *testing.T) {
	valid := func() ReplaceJournalEntryInput {
		return ReplaceJournalEntryInput{
			JournalEntryID: 1,
			OccurredAt:     time.Now(),
			Postings: []PostingInput{
				{LedgerAccountID: 1, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				{LedgerAccountID: 2, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			},
			Movements: []ReplaceMovementInput{
				{Kind: model.PositionMovementTrade, FinancialAccountID: 1, AssetID: 1, UnitsDelta: decimal.NewFromInt(3)},
			},
		}
	}

	tests := map[string]struct {
		mutate  func(*ReplaceJournalEntryInput)
		wantErr bool
	}{
		"a trade":       {mutate: func(*ReplaceJournalEntryInput) {}},
		"no movements":  {mutate: func(i *ReplaceJournalEntryInput) { i.Movements = nil }},
		"a vest":        {mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].Kind = model.PositionMovementVest }},
		"a sale":        {mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].UnitsDelta = decimal.NewFromInt(-3) }},
		"two movements": {mutate: func(i *ReplaceJournalEntryInput) { i.Movements = append(i.Movements, i.Movements[0]) }},
		"a cost basis with its currency": {mutate: func(i *ReplaceJournalEntryInput) {
			i.Movements[0].CostBasisAmount, i.Movements[0].CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(30)), ptr.Int64(1)
		}},
		"no entry":          {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.JournalEntryID = 0 }},
		"negative entry":    {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.JournalEntryID = -1 }},
		"no date":           {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.OccurredAt = time.Time{} }},
		"one posting":       {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Postings = i.Postings[:1] }},
		"a posting of zero": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Postings[0].Amount = decimal.Zero }},
		"an entry-less kind": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) {
			i.Movements[0].Kind = model.PositionMovementTransfer
		}},
		"no kind":              {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].Kind = "" }},
		"no financial account": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].FinancialAccountID = 0 }},
		"no asset":             {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].AssetID = 0 }},
		"zero units":           {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) { i.Movements[0].UnitsDelta = decimal.Zero }},
		"a cost basis without its currency": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) {
			i.Movements[0].CostBasisAmount = ptr.NewPtr(decimal.NewFromInt(30))
		}},
		"a currency without a cost basis": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) {
			i.Movements[0].CostBasisCurrencyID = ptr.Int64(1)
		}},
		"a cost basis in no currency": {wantErr: true, mutate: func(i *ReplaceJournalEntryInput) {
			i.Movements[0].CostBasisAmount, i.Movements[0].CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(30)), ptr.Int64(0)
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)

			err := input.Validate()

			if tc.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestReplaceJournalEntryInput_Entry(t *testing.T) {
	description := "corrected"
	input := ReplaceJournalEntryInput{
		JournalEntryID: 1, OccurredAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), Description: &description, Metadata: map[string]string{"k": "v"},
		Postings: []PostingInput{{LedgerAccountID: 1, Amount: decimal.NewFromInt(10), Side: model.PostingDebit}},
	}

	entry := input.Entry()

	require.Equal(t, PostJournalEntryInput{OccurredAt: input.OccurredAt, Description: &description, Metadata: input.Metadata, Postings: input.Postings}, entry)
}
