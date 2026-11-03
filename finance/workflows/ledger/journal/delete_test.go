package journal

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestDeleteJournalEntry() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{
			name:   "success",
			result: []any{&lt.DeleteJournalEntryResult{}, nil},
		},
		{
			name:   "success with warnings",
			result: []any{&lt.DeleteJournalEntryResult{Warnings: []lt.NegativeHolding{{FinancialAccountID: 1, FinancialAccountName: "Broker", AssetID: 2, AssetName: "ACME", At: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), Units: decimal.NewFromInt(-1)}}}, nil},
		},
		{
			name:    "activity failure",
			result:  []any{nil, assert.AnError},
			wantErr: true,
		},
	}

	input := lt.DeleteJournalEntryInput{JournalEntryID: 10}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ja := activity.Activities{}

			env.OnActivity(ja.DeleteJournalEntry, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(DeleteJournalEntry, input)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.DeleteJournalEntryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}
