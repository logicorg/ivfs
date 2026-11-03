package portfolio

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func ListPositionMovements(ctx workflow.Context, params lt.ListPositionMovementsParams) (*lt.ListPositionMovementsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result lt.ListPositionMovementsResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ListPositionMovements, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list position movements: %w", err)
	}

	return &result, nil
}
