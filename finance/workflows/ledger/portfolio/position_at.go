package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func PositionAt(ctx workflow.Context, params lt.PositionAtParams) (*lt.PositionAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result lt.PositionAtResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.PositionAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate position: %w", err)
	}

	return &result, nil
}
