package portfolio

import (
	"fmt"

	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func ListPositionMovements(ctx workflow.Context, params pft.ListPositionMovementsParams) (*pft.ListPositionMovementsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result pft.ListPositionMovementsResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ListPositionMovements, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list position movements: %w", err)
	}

	return &result, nil
}
