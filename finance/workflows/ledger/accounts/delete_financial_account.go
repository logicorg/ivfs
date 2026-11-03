package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteFinancialAccount deletes a financial account with its ledger
// accounts, as ADR 0012 describes. Its ledger accounts are closed and emptied
// together, so a transfer between two of them is deleted rather than kept,
// and a kept transfer, trade, or vest leg moves to the stray financial
// account. Split and adjustment legs on it are deleted, since each is a fact
// about this account only. Deleting an account that does not exist succeeds.
//
// It runs under DeleteFinancialAccountWorkflowID, and DeleteInstitution runs
// it as a child under the same ID.
func DeleteFinancialAccount(ctx workflow.Context, input lt.DeleteFinancialAccountInput) (*lt.DeleteAccountResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	if input.ReplacementID != nil {
		return nil, errMergeNotSupported
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	target := lt.DeletionTarget{FinancialAccountID: &input.FinancialAccountID}
	var closed lt.CloseAccountForDeletionResult
	if err := workflow.ExecuteActivity(ctx, la.CloseAccountForDeletion, target).Get(ctx, &closed); err != nil {
		return nil, fmt.Errorf("close financial account for deletion: %w", err)
	}
	if closed.Gone {
		return &lt.DeleteAccountResult{}, nil
	}

	// Entries go first: a trade's movement group is handled with its entry, so
	// the groups left afterwards are the ones without one.
	for {
		entryBatches, err := drainEntries(ctx, target)
		if err != nil {
			return nil, err
		}
		groupBatches, err := drainMovementGroups(ctx, target)
		if err != nil {
			return nil, err
		}
		if workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
			return nil, workflow.NewContinueAsNewError(ctx, DeleteFinancialAccount, input)
		}

		err = workflow.ExecuteActivity(ctx, la.DeleteFinancialAccount, input).Get(ctx, nil)
		if err == nil || workflows.HasErrorType(err, "ErrNotFound") {
			return &lt.DeleteAccountResult{}, nil
		}
		// A conflict after a pass that handled something is a line recorded
		// meanwhile, which the next pass handles. After an empty pass, nothing
		// will.
		if !workflows.HasErrorType(err, "ErrResourceConflict") || entryBatches+groupBatches == 0 {
			return nil, fmt.Errorf("delete financial account: %w", err)
		}
	}
}
