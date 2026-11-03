package workflows

import (
	"errors"
	"strconv"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DeleteLedgerAccountWorkflowID is the ID a ledger account's deletion runs
// under. Deletions are started with WorkflowIDConflictPolicy UseExisting, so a
// second request for the same account joins the running deletion instead of
// racing it (ADR 0012).
func DeleteLedgerAccountWorkflowID(id int64) string {
	return "delete-ledger-account-" + strconv.FormatInt(id, 10)
}

// DeleteFinancialAccountWorkflowID is the ID a financial account's deletion runs under.
func DeleteFinancialAccountWorkflowID(id int64) string {
	return "delete-financial-account-" + strconv.FormatInt(id, 10)
}

// DeleteInstitutionWorkflowID is the ID an institution's deletion runs under.
func DeleteInstitutionWorkflowID(id int64) string {
	return "delete-institution-" + strconv.FormatInt(id, 10)
}

// DeleteCategoryWorkflowID is the ID a category's deletion runs under. Its
// child runs under DeleteLedgerAccountWorkflowID.
func DeleteCategoryWorkflowID(id int64) string {
	return "delete-category-" + strconv.FormatInt(id, 10)
}

// ExecuteDeletionChild runs a deletion as a child workflow under workflowID,
// behind the prefix of the user the workflow runs as, which is the ID the
// gateway starts the same deletion under, and decodes its result into
// valuePtr.
//
// Child workflows take no ID conflict policy, so a child whose deletion is
// already running as a separate request fails to start with "already
// started". It then sleeps a minute and starts the child again: once the
// other run is done the account is gone, and the new child succeeds at once.
// Any other error is returned as is.
func ExecuteDeletionChild(ctx workflow.Context, workflowID string, wf, input, valuePtr any) error {
	uid, ok := tenancy.WorkflowUser(ctx)
	if !ok {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"user"}}
	}
	ctx = workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{WorkflowID: tenancy.WorkflowIDPrefix(uid) + workflowID})
	for {
		err := workflow.ExecuteChildWorkflow(ctx, wf, input).Get(ctx, valuePtr)
		var started *temporal.ChildWorkflowExecutionAlreadyStartedError
		if !errors.As(err, &started) {
			return err
		}
		if err := workflow.Sleep(ctx, time.Minute); err != nil {
			return err
		}
	}
}
