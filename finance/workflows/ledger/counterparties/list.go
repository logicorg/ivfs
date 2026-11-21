package counterparties

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ListCounterparties is a workflow that lists/searches counterparties.
func ListCounterparties(ctx workflow.Context, params lt.ListCounterpartiesParams) (*lt.ListCounterpartiesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ListCounterpartiesResult
	if err := workflow.ExecuteActivity(ctx, ca.ListCounterparties, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	return &result, nil
}
