// Package institution holds the workflows that manage institutions.
package institution

import (
	"fmt"

	institutionactivity "github.com/logicorg/ivfs/finance/activities/institution"
	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ia is a (nil) pointer to the institution activity struct. It is used to
// call the activity in the workflow.
var ia = (*institutionactivity.Institution)(nil)

// RetrieveInstitution is a workflow that retrieves an institution.
func RetrieveInstitution(ctx workflow.Context, params it.RetrieveInstitutionParams) (*it.RetrieveInstitutionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result it.RetrieveInstitutionResult
	if err := workflow.ExecuteActivity(ctx, ia.Retrieve, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve institution: %w", err)
	}
	return &result, nil
}
