package accounts

import (
	activity "github.com/logicorg/ivfs/finance/activities/ledger/accounts"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/workflow"
)

func (s *WorkflowUnitTestSuite) TestDeleteLedgerAccount() {
	conflict := internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "still referenced by ledger.posting.ledger_account_id"}

	tests := []struct {
		name  string
		input lt.DeleteLedgerAccountInput
		// closed is what CloseAccountForDeletion returns; nil skips it.
		closed   *lt.CloseAccountForDeletionResult
		closeErr error
		// passes holds, for each pass, the entries each of its batches handles
		// before the last one handles none.
		passes   [][]int
		batchErr error
		// deleteErrs is what the delete returns at the end of each pass.
		deleteErrs []error
		// continueAsNew is when Temporal suggests continuing as new: 0 never,
		// -1 from the start, n after the nth batch.
		continueAsNew int
		wantErr       string
	}{
		{
			name:    "an id that is not positive",
			input:   lt.DeleteLedgerAccountInput{},
			wantErr: "validation error",
		},
		{
			name:    "a replacement",
			input:   lt.DeleteLedgerAccountInput{LedgerAccountID: 7, ReplacementID: ptr.Int64(8)},
			wantErr: "merging an account into a replacement is not supported yet",
		},
		{
			name:   "an account that does not exist",
			input:  lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed: &lt.CloseAccountForDeletionResult{Gone: true},
		},
		{
			name:     "an account the close refuses",
			input:    lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:   &lt.CloseAccountForDeletionResult{},
			closeErr: internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account 7 has child accounts; delete or move them first"},
			wantErr:  "resource LedgerAccount conflict: ledger account 7 has child accounts; delete or move them first",
		},
		{
			name:       "an account without entries",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{}},
			deleteErrs: []error{nil},
		},
		{
			name:       "an account with entries",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{100, 100, 3}},
			deleteErrs: []error{nil},
		},
		{
			name:       "rows deleted meanwhile",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{3}},
			deleteErrs: []error{internalerrors.ErrNotFound{ResourceType: "LedgerAccount", ID: 7}},
		},
		{
			name:       "an entry recorded meanwhile",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{100}, {1}},
			deleteErrs: []error{conflict, nil},
		},
		{
			name:       "a conflict after a pass that handled nothing",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{100}, {}},
			deleteErrs: []error{conflict, conflict},
			wantErr:    "resource LedgerAccount conflict: still referenced by ledger.posting.ledger_account_id",
		},
		{
			name:       "a conflict on an account without entries",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{}},
			deleteErrs: []error{conflict},
			wantErr:    "resource LedgerAccount conflict: still referenced by ledger.posting.ledger_account_id",
		},
		{
			name:     "a batch that fails",
			input:    lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:   &lt.CloseAccountForDeletionResult{},
			batchErr: assert.AnError,
			wantErr:  "handle entries for deletion",
		},
		{
			name:       "a delete that fails",
			input:      lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     [][]int{{}},
			deleteErrs: []error{assert.AnError},
			wantErr:    "delete ledger account",
		},
		{
			name:          "continuing as new before a batch",
			input:         lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:        &lt.CloseAccountForDeletionResult{},
			continueAsNew: -1,
		},
		{
			name:          "continuing as new between two batches",
			input:         lt.DeleteLedgerAccountInput{LedgerAccountID: 7},
			closed:        &lt.CloseAccountForDeletionResult{},
			passes:        [][]int{{100}},
			continueAsNew: 1,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			target := lt.DeletionTarget{LedgerAccountID: &tc.input.LedgerAccountID}
			batch := lt.DeletionBatchInput{Target: target, Limit: deletionBatchSize}

			if tc.closed != nil {
				env.OnActivity(aa.CloseAccountForDeletion, mock.Anything, target).Return(tc.closed, tc.closeErr).Once()
			}
			if tc.batchErr != nil {
				// Not Once: a retryable error is retried.
				env.OnActivity(aa.HandleEntriesForDeletion, mock.Anything, batch).Return(nil, tc.batchErr)
			}
			batches := 0
			suggested := func() bool { return tc.continueAsNew > 0 && batches == tc.continueAsNew }
			for n, pass := range tc.passes {
				for _, handled := range pass {
					call := env.OnActivity(aa.HandleEntriesForDeletion, mock.Anything, batch).Return(&lt.DeletionBatchResult{Handled: handled}, nil).Once()
					batches++
					if suggested() {
						call.Run(func(mock.Arguments) { env.SetContinueAsNewSuggested(true) })
						break
					}
				}
				if suggested() {
					break
				}
				env.OnActivity(aa.HandleEntriesForDeletion, mock.Anything, batch).Return(&lt.DeletionBatchResult{}, nil).Once()
				deleted := env.OnActivity(aa.DeleteLedgerAccount, mock.Anything, tc.input).Return(tc.deleteErrs[n])
				if tc.deleteErrs[n] != assert.AnError {
					deleted.Once()
				}
			}
			if tc.continueAsNew == -1 {
				env.SetContinueAsNewSuggested(true)
			}

			env.ExecuteWorkflow(DeleteLedgerAccount, tc.input)

			env.AssertExpectations(t)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			if tc.continueAsNew != 0 {
				var continued *workflow.ContinueAsNewError
				require.ErrorAs(t, env.GetWorkflowError(), &continued)
				require.Equal(t, "DeleteLedgerAccount", continued.WorkflowType.Name)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var result lt.DeleteAccountResult
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Empty(t, result.Warnings)
		})
	}
}
