package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func RegisterPositionMovement(ctx workflow.Context, input lt.RegisterPositionMovementInput) (*lt.RegisterPositionMovementResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)
	var result lt.RegisterPositionMovementResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RegisterPositionMovement, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register position movement: %w", err)
	}

	return &result, nil
}
