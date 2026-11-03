package workflows

import (
	"testing"
	"time"

	"github.com/google/uuid"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/converter"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestDeletionWorkflowIDs(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "ledger account", got: DeleteLedgerAccountWorkflowID(7), want: "delete-ledger-account-7"},
		{name: "financial account", got: DeleteFinancialAccountWorkflowID(7), want: "delete-financial-account-7"},
		{name: "institution", got: DeleteInstitutionWorkflowID(7), want: "delete-institution-7"},
		{name: "category", got: DeleteCategoryWorkflowID(7), want: "delete-category-7"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got)
		})
	}
}

// deletionChild stands for a deletion workflow; the tests mock it.
func deletionChild(workflow.Context, int64) (int64, error) { return 0, nil }

func deletionParent(ctx workflow.Context, id int64) (int64, error) {
	var result int64
	err := ExecuteDeletionChild(ctx, "delete-test-7", deletionChild, id, &result)
	return result, err
}

// TestExecuteDeletionChild checks that the child runs under the user's
// prefix, so it collides with the same user's deletion and no one else's,
// and that a parent running as no one starts no child.
func TestExecuteDeletionChild(t *testing.T) {
	user := uuid.MustParse("0192f3a0-0000-7000-8000-000000000001")
	tests := []struct {
		name string
		// noUser runs the parent as no one.
		noUser bool
		// startErrs is what each start of the child fails with, before the
		// last one runs it.
		startErrs []error
		childErr  error
		// wantWait is the least time the parent spends waiting to start the
		// child again.
		wantWait time.Duration
		wantErr  string
	}{
		{name: "a child that starts"},
		{name: "a parent that runs as no one", noUser: true, wantErr: "missing required parameters: user"},
		{
			name:      "a child already running",
			startErrs: []error{&temporal.ChildWorkflowExecutionAlreadyStartedError{}, &temporal.ChildWorkflowExecutionAlreadyStartedError{}},
			wantWait:  2 * time.Minute,
		},
		{
			name:     "a child that fails",
			childErr: internalerrors.ErrResourceConflict{ResourceType: "LedgerAccount", Reason: "still referenced by ledger.posting.ledger_account_id"},
			wantErr:  "resource LedgerAccount conflict: still referenced by ledger.posting.ledger_account_id",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var suite testsuite.WorkflowTestSuite
			env := suite.NewTestWorkflowEnvironment()
			env.RegisterWorkflow(deletionChild)
			if !tc.noUser {
				temporaltesting.RunAs(t, env, user)
				for _, err := range tc.startErrs {
					env.OnWorkflow(deletionChild, mock.Anything, int64(7)).Return(int64(0), err).Once()
				}
				env.OnWorkflow(deletionChild, mock.Anything, int64(7)).Return(int64(7), tc.childErr).Once()
			}
			var ids []string
			env.SetOnChildWorkflowStartedListener(func(info *workflow.Info, _ workflow.Context, _ converter.EncodedValues) {
				ids = append(ids, info.WorkflowExecution.ID)
			})
			start := env.Now()

			env.ExecuteWorkflow(deletionParent, int64(7))

			env.AssertExpectations(t)
			require.GreaterOrEqual(t, env.Now().Sub(start), tc.wantWait)
			if !tc.noUser {
				require.Len(t, ids, len(tc.startErrs)+1)
			}
			for _, id := range ids {
				require.Equal(t, tenancy.WorkflowIDPrefix(user)+"delete-test-7", id)
			}
			if tc.wantErr != "" {
				require.ErrorContains(t, env.GetWorkflowError(), tc.wantErr)
				return
			}
			require.NoError(t, env.GetWorkflowError())
			var result int64
			require.NoError(t, env.GetWorkflowResult(&result))
			require.Equal(t, int64(7), result)
		})
	}
}
