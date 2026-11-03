package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteLedgerAccount deletes a ledger account of any kind, as ADR 0012
// describes. Entries that touch only the account and what explains it, such as
// categories and equity, are deleted with it; entries that also touch a
// surviving balance are kept, with the account's lines moved to the stray
// accounts. Deleting an account that does not exist succeeds.
//
// It runs under DeleteLedgerAccountWorkflowID, and DeleteCategory runs it as a
// child under the same ID, so one account is never emptied twice at once.
func DeleteLedgerAccount(ctx workflow.Context, input lt.DeleteLedgerAccountInput) (*lt.DeleteAccountResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.ReplacementID != nil {
		return nil, errMergeNotSupported
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	target := lt.DeletionTarget{LedgerAccountID: &input.LedgerAccountID}
	var closed lt.CloseAccountForDeletionResult
	if err := workflow.ExecuteActivity(ctx, la.CloseAccountForDeletion, target).Get(ctx, &closed); err != nil {
		return nil, fmt.Errorf("close ledger account for deletion: %w", err)
	}
	if closed.Gone {
		return &lt.DeleteAccountResult{}, nil
	}

	// A ledger account has no movements, so its entries are all there is to
	// handle.
	for {
		batches, err := drainEntries(ctx, target)
		if err != nil {
			return nil, err
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			return nil, workflow.NewContinueAsNewError(ctx, DeleteLedgerAccount, input)
		}

		err = workflow.ExecuteActivity(ctx, la.DeleteLedgerAccount, input).Get(ctx, nil)
		if err == nil || workflows.HasErrorType(err, "ErrNotFound") {
			return &lt.DeleteAccountResult{}, nil
		}
		// A conflict after a pass that handled something is an entry recorded
		// meanwhile, which the next pass handles. After an empty pass, nothing
		// will.
		if !workflows.HasErrorType(err, "ErrResourceConflict") || batches == 0 {
			return nil, fmt.Errorf("delete ledger account: %w", err)
		}
	}
}
