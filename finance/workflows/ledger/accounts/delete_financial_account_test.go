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

// financialAccountPass is what the batches of one pass handle before the last
// one of each kind handles nothing.
type financialAccountPass struct {
	entries, groups []int
}

func (s *WorkflowUnitTestSuite) TestDeleteFinancialAccount() {
	conflict := internalerrors.ErrResourceConflict{ResourceType: "FinancialAccount", Reason: "still referenced by asset.position_movement.financial_account_id"}

	tests := []struct {
		name  string
		input lt.DeleteFinancialAccountInput
		// closed is what CloseAccountForDeletion returns; nil skips it.
		closed   *lt.CloseAccountForDeletionResult
		closeErr error
		passes   []financialAccountPass
		// entryErr and groupErr fail the first batch of their kind.
		entryErr, groupErr error
		// deleteErrs is what the delete returns at the end of each pass.
		deleteErrs []error
		// continueAsNew is when Temporal suggests continuing as new: 0 never,
		// -1 from the start, n after the nth batch of either kind.
		continueAsNew int
		wantErr       string
	}{
		{
			name:    "an id that is not positive",
			input:   lt.DeleteFinancialAccountInput{},
			wantErr: "validation error",
		},
		{
			name:    "a replacement",
			input:   lt.DeleteFinancialAccountInput{FinancialAccountID: 7, ReplacementID: ptr.Int64(8)},
			wantErr: "merging an account into a replacement is not supported yet",
		},
		{
			name:   "an account that does not exist",
			input:  lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed: &lt.CloseAccountForDeletionResult{Gone: true},
		},
		{
			name:     "an account the close refuses",
			input:    lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:   &lt.CloseAccountForDeletionResult{},
			closeErr: internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "ledger account 9 has child accounts outside it; delete or move them first"},
			wantErr:  "resource LedgerAccount conflict: ledger account 9 has child accounts outside it",
		},
		{
			name:       "an empty account",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{}},
			deleteErrs: []error{nil},
		},
		{
			name:       "an account with entries and movement groups",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{entries: []int{100, 4}, groups: []int{100, 100, 2}}},
			deleteErrs: []error{nil},
		},
		{
			name:       "rows deleted meanwhile",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{entries: []int{3}}},
			deleteErrs: []error{internalerrors.ErrNotFound{ResourceType: "FinancialAccount", ID: 7}},
		},
		{
			name:       "a movement recorded meanwhile",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{entries: []int{100}}, {groups: []int{1}}},
			deleteErrs: []error{conflict, nil},
		},
		{
			name:       "a conflict after a pass that handled only groups",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{groups: []int{5}}, {}},
			deleteErrs: []error{conflict, conflict},
			wantErr:    "resource FinancialAccount conflict: still referenced by asset.position_movement.financial_account_id",
		},
		{
			name:       "a conflict on an empty account",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{}},
			deleteErrs: []error{conflict},
			wantErr:    "resource FinancialAccount conflict: still referenced by asset.position_movement.financial_account_id",
		},
		{
			name:     "an entry batch that fails",
			input:    lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:   &lt.CloseAccountForDeletionResult{},
			entryErr: assert.AnError,
			wantErr:  "handle entries for deletion",
		},
		{
			name:     "a movement group batch that fails",
			input:    lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:   &lt.CloseAccountForDeletionResult{},
			passes:   []financialAccountPass{{entries: []int{2}}},
			groupErr: assert.AnError,
			wantErr:  "handle movement groups for deletion",
		},
		{
			name:       "a delete that fails",
			input:      lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:     &lt.CloseAccountForDeletionResult{},
			passes:     []financialAccountPass{{}},
			deleteErrs: []error{assert.AnError},
			wantErr:    "delete financial account",
		},
		{
			name:          "continuing as new before a batch",
			input:         lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:        &lt.CloseAccountForDeletionResult{},
			continueAsNew: -1,
		},
		{
			name:          "continuing as new among the entries",
			input:         lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:        &lt.CloseAccountForDeletionResult{},
			passes:        []financialAccountPass{{entries: []int{100}}},
			continueAsNew: 1,
		},
		{
			name:          "continuing as new among the movement groups",
			input:         lt.DeleteFinancialAccountInput{FinancialAccountID: 7},
			closed:        &lt.CloseAccountForDeletionResult{},
			passes:        []financialAccountPass{{entries: []int{100}, groups: []int{100}}},
			continueAsNew: 2,
		},
	}
	for _, tc := range tests {
		s.Run(tc.name, func() {
			t := s.T()
			env := s.NewTestWorkflowEnvironment()
			aa := activity.Activities{}
			target := lt.DeletionTarget{FinancialAccountID: &tc.input.FinancialAccountID}
			batch := lt.DeletionBatchInput{Target: target, Limit: deletionBatchSize}

			if tc.closed != nil {
				env.OnActivity(aa.CloseAccountForDeletion, mock.Anything, target).Return(tc.closed, tc.closeErr).Once()
			}
			// Not Once: a retryable error is retried.
			if tc.entryErr != nil {
				env.OnActivity(aa.HandleEntriesForDeletion, mock.Anything, batch).Return(nil, tc.entryErr)
			}
			if tc.groupErr != nil {
				env.OnActivity(aa.HandleMovementGroupsForDeletion, mock.Anything, batch).Return(nil, tc.groupErr)
			}

			batches := 0
			suggested := func() bool { return tc.continueAsNew > 0 && batches == tc.continueAsNew }
			// expect mocks the batches of one kind and the empty one that ends
			// them, and reports whether Temporal suggested continuing as new.
			expect := func(activity any, handled []int) bool {
				for _, n := range handled {
					call := env.OnActivity(activity, mock.Anything, batch).Return(&lt.DeletionBatchResult{Handled: n}, nil).Once()
					batches++
					if suggested() {
						call.Run(func(mock.Arguments) { env.SetContinueAsNewSuggested(true) })
						return true
					}
				}
				env.OnActivity(activity, mock.Anything, batch).Return(&lt.DeletionBatchResult{}, nil).Once()
				return false
			}
			for n, pass := range tc.passes {
				if expect(aa.HandleEntriesForDeletion, pass.entries) || tc.groupErr != nil {
					break
				}
				if expect(aa.HandleMovementGroupsForDeletion, pass.groups) {
					break
				}
				deleted := env.OnActivity(aa.DeleteFinancialAccount, mock.Anything, tc.input).Return(tc.deleteErrs[n])
				if tc.deleteErrs[n] != assert.AnError {
					deleted.Once()
				}
			}
			if tc.continueAsNew == -1 {
				env.SetContinueAsNewSuggested(true)
			}

			env.ExecuteWorkflow(DeleteFinancialAccount, tc.input)

			env.AssertExpectations(t)
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			if tc.continueAsNew != 0 {
				var continued *workflow.ContinueAsNewError
				require.ErrorAs(t, env.GetWorkflowError(), &continued)
				require.Equal(t, "DeleteFinancialAccount", continued.WorkflowType.Name)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var result lt.DeleteAccountResult
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Empty(t, result.Warnings)
		})
	}
}
