package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func RegisterVest(ctx workflow.Context, input lt.RegisterVestInput) (*lt.RegisterVestResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.Entry.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.Entry.IdempotencyKey)
	var result lt.RegisterVestResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RegisterVest, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register vest: %w", err)
	}

	return &result, nil
}
