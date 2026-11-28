package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// MoveCategory moves a category under another parent, or to the top level.
func MoveCategory(ctx workflow.Context, input ct.MoveCategoryInput) (*ct.CategoryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.CategoryResult
	if err := workflow.ExecuteActivity(ctx, ca.MoveCategory, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("move category: %w", err)
	}
	return &result, nil
}
