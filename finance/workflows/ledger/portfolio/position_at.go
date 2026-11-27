package portfolio

import (
	"fmt"

	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func PositionAt(ctx workflow.Context, params pft.PositionAtParams) (*pft.PositionAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result pft.PositionAtResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.PositionAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate position: %w", err)
	}

	return &result, nil
}
