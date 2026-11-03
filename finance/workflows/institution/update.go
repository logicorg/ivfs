package institution

import (
	"fmt"
	"strings"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateInstitution is a workflow that renames an institution and/or updates
// its kind. It refuses to change the "Untracked" institution, or to rename
// another one onto its name.
func UpdateInstitution(ctx workflow.Context, params it.UpdateInstitutionParams) (*it.UpdateInstitutionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	institution, err := retrieveInstitution(ctx, params.FindBy)
	if err != nil {
		return nil, fmt.Errorf("update institution: %w", err)
	}
	if err := refuseUntrackedInstitution(institution, "updated"); err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(params.Name); name != "" {
		if err := refuseUntrackedName(institution.Name, name, fmt.Sprintf("renamed to %q", name)); err != nil {
			return nil, err
		}
	}

	var result it.UpdateInstitutionResult
	if err := workflow.ExecuteActivity(ctx, ia.Update, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("update institution: %w", err)
	}
	return &result, nil
}
