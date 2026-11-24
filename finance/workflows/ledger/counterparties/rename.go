package counterparties

import (
	"fmt"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RenameCounterparty is a workflow that renames a counterparty.
func RenameCounterparty(ctx workflow.Context, input cpt.RenameCounterpartyInput) (*cpt.CounterpartyMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result cpt.CounterpartyMutationResult
	if err := workflow.ExecuteActivity(ctx, ca.RenameCounterparty, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("rename counterparty: %w", err)
	}
	return &result, nil
}
