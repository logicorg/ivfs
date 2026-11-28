package counterparties

import (
	"fmt"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ListCounterparties is a workflow that lists/searches counterparties.
func ListCounterparties(ctx workflow.Context, params cpt.ListCounterpartiesParams) (*cpt.ListCounterpartiesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result cpt.ListCounterpartiesResult
	if err := workflow.ExecuteActivity(ctx, ca.ListCounterparties, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list counterparties: %w", err)
	}
	return &result, nil
}
