package counterparties

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/counterparties"
	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (s *WorkflowUnitTestSuite) TestDeleteCounterparty() {
	conflict := internalerrors.ErrResourceConflict{ResourceType: "Counterparty", Reason: "still referenced by ledger.journal_entry.counterparty_id"}

	tests := []struct {
		name  string
		input cpt.DeleteCounterpartyInput
		// cleared and deleteErrs are what the clear and the delete return in
		// each round.
		cleared    []int64
		deleteErrs []error
		// clearErr fails the first clear.
		clearErr    error
		wantCleared int64
		wantErr     string
	}{
		{
			name:    "an id that is not positive",
			input:   cpt.DeleteCounterpartyInput{},
			wantErr: "validation error",
		},
		{
			name:        "deleted in one round",
			input:       cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			cleared:     []int64{4},
			deleteErrs:  []error{nil},
			wantCleared: 4,
		},
		{
			name:        "a counterparty nothing names",
			input:       cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			cleared:     []int64{0},
			deleteErrs:  []error{nil},
			wantCleared: 0,
		},
		{
			name:        "an entry recorded between the clear and the delete",
			input:       cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			cleared:     []int64{4, 1},
			deleteErrs:  []error{conflict, nil},
			wantCleared: 5,
		},
		{
			name:       "a conflict in the last round",
			input:      cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			cleared:    []int64{4, 1},
			deleteErrs: []error{conflict, conflict},
			wantErr:    "resource Counterparty conflict: still referenced by ledger.journal_entry.counterparty_id",
		},
		{
			name:       "a delete that fails otherwise",
			input:      cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			cleared:    []int64{4},
			deleteErrs: []error{assert.AnError},
			wantErr:    "delete counterparty",
		},
		{
			name:     "a clear that fails",
			input:    cpt.DeleteCounterpartyInput{CounterpartyID: 7},
			clearErr: assert.AnError,
			wantErr:  "clear counterparty references",
		},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			clearInput := cpt.ClearCounterpartyReferencesInput{CounterpartyID: tc.input.CounterpartyID}

			if tc.clearErr != nil {
				// Not Once: a retryable error is retried.
				env.OnActivity(aa.ClearCounterpartyReferences, mock.Anything, clearInput).Return(nil, tc.clearErr)
			}
			for round, cleared := range tc.cleared {
				env.OnActivity(aa.ClearCounterpartyReferences, mock.Anything, clearInput).
					Return(&cpt.ClearCounterpartyReferencesResult{Cleared: cleared}, nil).Once()
				deleted := env.OnActivity(aa.DeleteCounterparty, mock.Anything, tc.input).Return(tc.deleteErrs[round])
				if tc.deleteErrs[round] != assert.AnError {
					deleted.Once()
				}
			}

			env.ExecuteWorkflow(DeleteCounterparty, tc.input)

			env.AssertExpectations(t)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var result cpt.DeleteCounterpartyResult
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, cpt.DeleteCounterpartyResult{Cleared: tc.wantCleared}, result)
		})
	}
}
