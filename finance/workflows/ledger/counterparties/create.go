package counterparties

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CreateCounterparty is a workflow that creates a counterparty.
func CreateCounterparty(ctx workflow.Context, input lt.CreateCounterpartyInput) (*lt.CreateCounterpartyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.CreateCounterpartyResult
	if err := workflow.ExecuteActivity(ctx, ca.CreateCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create counterparty: %w", err)
	}
	return &result, nil
}
