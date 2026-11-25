package portfolio

import (
	"fmt"

	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func RegisterPositionMovement(ctx workflow.Context, input pft.RegisterPositionMovementInput) (*pft.RegisterPositionMovementResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)
	var result pft.RegisterPositionMovementResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RegisterPositionMovement, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register position movement: %w", err)
	}

	return &result, nil
}
