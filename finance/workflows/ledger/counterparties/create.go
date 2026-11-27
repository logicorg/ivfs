package counterparties

import (
	"fmt"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CreateCounterparty is a workflow that creates a counterparty.
func CreateCounterparty(ctx workflow.Context, input cpt.CreateCounterpartyInput) (*cpt.CreateCounterpartyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result cpt.CreateCounterpartyResult
	if err := workflow.ExecuteActivity(ctx, ca.CreateCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create counterparty: %w", err)
	}
	return &result, nil
}
