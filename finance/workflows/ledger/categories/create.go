package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CreateCategory creates a category, or finds the one with its name and kind
// under the same parent.
func CreateCategory(ctx workflow.Context, input ct.CreateCategoryInput) (*ct.CategoryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.CategoryResult
	if err := workflow.ExecuteActivity(ctx, ca.CreateCategory, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}
	return &result, nil
}
