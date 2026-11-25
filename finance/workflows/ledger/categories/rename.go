package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RenameCategory renames a category.
func RenameCategory(ctx workflow.Context, input ct.RenameCategoryInput) (*ct.CategoryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.CategoryResult
	if err := workflow.ExecuteActivity(ctx, ca.RenameCategory, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("rename category: %w", err)
	}
	return &result, nil
}
