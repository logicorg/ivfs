package journal

import (
	"testing"
	"time"

	"github.com/google/uuid"
	activity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.temporal.io/sdk/testsuite"
)

type WorkflowUnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite
}

func TestWorkflowUnitTestSuite(t *testing.T) { suite.Run(t, new(WorkflowUnitTestSuite)) }

func (s *WorkflowUnitTestSuite) TestPostJournalEntry() {
	input := lt.PostJournalEntryInput{IdempotencyKey: "entry-1", OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	tests := []struct {
		name    string
		result  *lt.PostJournalEntryResult
		wantErr bool
	}{
		{name: "success", result: &lt.PostJournalEntryResult{JournalEntryID: 10}},
		{name: "activity failure", wantErr: true},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			env := s.NewTestWorkflowEnvironment()
			ja := activity.Activities{}
			if tc.wantErr {
				env.OnActivity(ja.PostJournalEntry, mock.Anything, input).Return(nil, assert.AnError)
			} else {
				env.OnActivity(ja.PostJournalEntry, mock.Anything, input).Return(tc.result, nil)
			}
			env.ExecuteWorkflow(PostJournalEntry, input)
			if tc.wantErr {
				require.Error(s.T(), env.GetWorkflowError())
				return
			}
			require.NoError(s.T(), env.GetWorkflowError())
			var result *lt.PostJournalEntryResult
			require.NoError(s.T(), env.GetWorkflowResult(&result))
			assert.Equal(s.T(), tc.result, result)
		})
	}
}

// TestPostJournalEntryDefaultsIdempotencyKey checks that a caller leaving
// IdempotencyKey empty gets a generated one filled in before the activity is
// called, and that two independent executions produce two different
// generated keys.
func (s *WorkflowUnitTestSuite) TestPostJournalEntryDefaultsIdempotencyKey() {
	input := lt.PostJournalEntryInput{OccurredAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		ja := activity.Activities{}

		env.OnActivity(ja.PostJournalEntry, mock.Anything, mock.MatchedBy(func(got lt.PostJournalEntryInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&lt.PostJournalEntryResult{JournalEntryID: 1}, nil)

		env.ExecuteWorkflow(PostJournalEntry, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}

func (s *WorkflowUnitTestSuite) TestRetrieveJournalEntry() {
	input := lt.RetrieveJournalEntryParams{ID: 10}
	expected := &lt.RetrieveJournalEntryResult{}
	env := s.NewTestWorkflowEnvironment()
	ja := activity.Activities{}
	env.OnActivity(ja.RetrieveJournalEntry, mock.Anything, input).Return(expected, nil)
	env.ExecuteWorkflow(RetrieveJournalEntry, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.RetrieveJournalEntryResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	assert.Equal(s.T(), expected, result)
}

func (s *WorkflowUnitTestSuite) TestReverseJournalEntry() {
	input := lt.ReverseJournalEntryInput{JournalEntryID: 10, IdempotencyKey: "reverse-10"}
	env := s.NewTestWorkflowEnvironment()
	ja := activity.Activities{}
	env.OnActivity(ja.ReverseJournalEntry, mock.Anything, input).Return(&lt.ReverseJournalEntryResult{JournalEntryID: 11}, nil)
	env.ExecuteWorkflow(ReverseJournalEntry, input)
	require.NoError(s.T(), env.GetWorkflowError())
	var result *lt.ReverseJournalEntryResult
	require.NoError(s.T(), env.GetWorkflowResult(&result))
	require.Equal(s.T(), int64(11), result.JournalEntryID)
}

// TestReverseJournalEntryDefaultsIdempotencyKey checks that a caller leaving
// IdempotencyKey empty gets a generated one filled in before the activity is
// called, and that two independent executions produce two different
// generated keys.
func (s *WorkflowUnitTestSuite) TestReverseJournalEntryDefaultsIdempotencyKey() {
	input := lt.ReverseJournalEntryInput{JournalEntryID: 10}

	var generated [2]string
	for i := range generated {
		env := s.NewTestWorkflowEnvironment()
		ja := activity.Activities{}

		env.OnActivity(ja.ReverseJournalEntry, mock.Anything, mock.MatchedBy(func(got lt.ReverseJournalEntryInput) bool {
			generated[i] = got.IdempotencyKey
			return true
		})).Return(&lt.ReverseJournalEntryResult{JournalEntryID: 11}, nil)

		env.ExecuteWorkflow(ReverseJournalEntry, input)
		require.NoError(s.T(), env.GetWorkflowError())
	}

	for _, key := range generated {
		_, err := uuid.Parse(key)
		require.NoError(s.T(), err)
	}
	require.NotEqual(s.T(), generated[0], generated[1])
}
