package journal

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestSetJournalEntryTags() {
	tests := []struct {
		name    string
		result  []any
		wantErr bool
	}{
		{name: "success", result: []any{&jt.SetJournalEntryTagsResult{TagIDs: []int64{2, 3}}, nil}},
		{name: "activity failure", result: []any{nil, assert.AnError}, wantErr: true},
	}

	input := jt.SetJournalEntryTagsInput{JournalEntryID: 10, TagIDs: []int64{3, 2}}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ja := activity.Activities{}

			env.OnActivity(ja.SetJournalEntryTags, mock.Anything, input).Return(tc.result...)

			env.ExecuteWorkflow(SetJournalEntryTags, input)

			if tc.wantErr {
				require.ErrorContains(s.T(), env.GetWorkflowError(), "set journal entry tags")
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *jt.SetJournalEntryTagsResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result[0], result)
		})
	}
}
