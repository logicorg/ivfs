package journal

import (
	"time"

	activity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestListJournalEntries() {
	page := &lt.ListJournalEntriesResult{
		Entries: []model.JournalEntry{{ID: 10, Status: model.JournalEntryPosted}},
		Next:    &lt.JournalEntryCursor{OccurredAt: time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC), ID: 10},
	}

	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{name: "success", result: []any{page, nil}},
		{name: "activity failure", result: []any{nil, assert.AnError}, wantErr: true},
	}

	params := lt.ListJournalEntriesParams{LedgerAccountID: 1, Descending: true, Limit: 1}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ja := activity.Activities{}

			env.OnActivity(ja.ListJournalEntries, mock.Anything, params).Return(tc.result...)

			env.ExecuteWorkflow(ListJournalEntries, params)

			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.ListJournalEntriesResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			require.Len(s.T(), result.Entries, 1)
			assert.Equal(s.T(), page.Entries[0].ID, result.Entries[0].ID)
			assert.Equal(s.T(), page.Next, result.Next)
		})
	}
}
