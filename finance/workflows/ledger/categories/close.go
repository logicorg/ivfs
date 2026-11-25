package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CloseCategory closes a category, which only stops it from being offered.
func CloseCategory(ctx workflow.Context, input ct.CloseCategoryInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	if err := workflow.ExecuteActivity(ctx, ca.CloseCategory, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("close category: %w", err)
	}
	return nil
}
