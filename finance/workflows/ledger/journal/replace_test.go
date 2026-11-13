package journal

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestReplaceJournalEntry() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.ReplaceJournalEntryResult{JournalEntryID: 10}, nil},
		},
		{
			name: "success with movements and warnings",
			result: []any{&lt.ReplaceJournalEntryResult{
				JournalEntryID:      10,
				PositionMovementIDs: []int64{7},
				Warnings:            []lt.NegativeHolding{{FinancialAccountID: 1, FinancialAccountName: "Broker", AssetID: 2, AssetName: "ACME", At: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-1)}},
			}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.ReplaceJournalEntryInput{
		JournalEntryID: 10,
		OccurredAt:     time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Postings: []lt.PostingInput{
			{LedgerAccountID: 1, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
			{LedgerAccountID: 2, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		},
		Movements: []lt.ReplaceMovementInput{{Kind: model.PositionMovementTrade, FinancialAccountID: 1, AssetID: 2, UnitsDelta: decimal.NewFromInt(3)}},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ja := activity.Activities{}

			env.OnActivity(ja.ReplaceJournalEntry, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(ReplaceJournalEntry, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.ReplaceJournalEntryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}
