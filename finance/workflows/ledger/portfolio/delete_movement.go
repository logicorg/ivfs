package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeletePositionMovement deletes an entry-less movement group and its reversals
// in one atomic activity. The activity is idempotent, so a retry needs no key.
func DeletePositionMovement(ctx workflow.Context, input lt.DeletePositionMovementInput) (*lt.DeletePositionMovementResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.DeletePositionMovementResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.DeletePositionMovement, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("delete position movement: %w", err)
	}

	return &result, nil
}
