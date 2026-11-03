package institution

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// InsertInstitution is a workflow that creates an institution. It refuses
// the name of the "Untracked" institution, which only the ledger creates.
func InsertInstitution(ctx workflow.Context, params it.InsertInstitutionParams) (*it.InsertInstitutionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	if err := refuseUntrackedName(params.Name, params.Name, "created"); err != nil {
		return nil, err
	}

	var result it.InsertInstitutionResult
	if err := workflow.ExecuteActivity(ctx, ia.Insert, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("insert institution: %w", err)
	}
	return &result, nil
}
