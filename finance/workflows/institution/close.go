package institution

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CloseInstitution is a workflow that closes an institution. It refuses to
// close the "Untracked" institution, since every later deletion that moves
// units onto its account would fail on it.
func CloseInstitution(ctx workflow.Context, params it.CloseParams) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	institution, err := retrieveInstitution(ctx, params.FindBy)
	if err != nil {
		return fmt.Errorf("close institution: %w", err)
	}
	if err := refuseUntrackedInstitution(institution, "closed"); err != nil {
		return err
	}

	if err := workflow.ExecuteActivity(ctx, ia.Close, params).Get(ctx, nil); err != nil {
		return fmt.Errorf("close institution: %w", err)
	}
	return nil
}
