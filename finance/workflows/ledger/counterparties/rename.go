package counterparties

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RenameCounterparty is a workflow that renames a counterparty.
func RenameCounterparty(ctx workflow.Context, input lt.RenameCounterpartyInput) (*lt.CounterpartyMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.CounterpartyMutationResult
	if err := workflow.ExecuteActivity(ctx, ca.RenameCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("rename counterparty: %w", err)
	}
	return &result, nil
}
