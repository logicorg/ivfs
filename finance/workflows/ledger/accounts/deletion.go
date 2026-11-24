package accounts

import (
	"fmt"
	act "github.com/logicorg/ivfs/finance/common/types/accounttypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"go.temporal.io/sdk/workflow"
)

// deletionBatchSize is how many entries, or movement groups, one batch
// activity handles in its transaction.
const deletionBatchSize = 100

// errMergeNotSupported refuses a replacement until merging is implemented.
var errMergeNotSupported = internalerrors.ErrInvalidParameter{
	Description: "merging an account into a replacement is not supported yet; delete it without a replacement",
}

// drainEntries runs the entry batch activity on target until a batch handles
// nothing or Temporal suggests continuing as new, and returns how many batches
// handled something. Each batch is one transaction that leaves the ledger
// balanced, so the workflow can stop between two of them and pick up where it
// stopped.
func drainEntries(ctx workflow.Context, target act.DeletionTarget) (int, error) {
	batches, err := drain(ctx, da.HandleEntriesForDeletion, target)
	if err != nil {
		return batches, fmt.Errorf("handle entries for deletion: %w", err)
	}
	return batches, nil
}

// drainMovementGroups is drainEntries for the movement groups left on target
// once its entries are gone.
func drainMovementGroups(ctx workflow.Context, target act.DeletionTarget) (int, error) {
	batches, err := drain(ctx, da.HandleMovementGroupsForDeletion, target)
	if err != nil {
		return batches, fmt.Errorf("handle movement groups for deletion: %w", err)
	}
	return batches, nil
}

func drain(ctx workflow.Context, batch any, target act.DeletionTarget) (int, error) {
	batches := 0
	for !workflow.GetInfo(ctx).GetContinueAsNewSuggested() {
		var result act.DeletionBatchResult
		if err := workflow.ExecuteActivity(ctx, batch, act.DeletionBatchInput{Target: target, Limit: deletionBatchSize}).Get(ctx, &result); err != nil {
			return batches, err
		}
		if result.Handled == 0 {
			break
		}
		batches++
	}
	return batches, nil
}
