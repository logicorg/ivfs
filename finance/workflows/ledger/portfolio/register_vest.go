package portfolio

import (
	"fmt"

	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func RegisterVest(ctx workflow.Context, input pft.RegisterVestInput) (*pft.RegisterVestResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.Entry.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.Entry.IdempotencyKey)
	var result pft.RegisterVestResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RegisterVest, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register vest: %w", err)
	}

	return &result, nil
}
