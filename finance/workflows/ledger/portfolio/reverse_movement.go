package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func ReversePositionMovement(ctx workflow.Context, input lt.ReversePositionMovementInput) (*lt.ReversePositionMovementResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)
	var result lt.ReversePositionMovementResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ReversePositionMovement, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("reverse position movement: %w", err)
	}

	return &result, nil
}
