package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteCategory deletes a category, or merges it into a replacement. No
// entry is deleted.
func DeleteCategory(ctx workflow.Context, input ct.DeleteCategoryInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	if err := workflow.ExecuteActivity(ctx, ca.DeleteCategory, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("delete category: %w", err)
	}
	return nil
}
