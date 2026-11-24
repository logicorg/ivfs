package counterparties

import (
	"fmt"

	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RetrieveCounterparty is a workflow that retrieves a counterparty.
func RetrieveCounterparty(ctx workflow.Context, params cpt.RetrieveCounterpartyParams) (*cpt.RetrieveCounterpartyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result cpt.RetrieveCounterpartyResult
	if err := workflow.ExecuteActivity(ctx, ca.RetrieveCounterparty, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve counterparty: %w", err)
	}
	return &result, nil
}
