package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// PreviewCategoryDeletion reports what DeleteCategory would change, and
// changes nothing.
func PreviewCategoryDeletion(ctx workflow.Context, input ct.PreviewCategoryDeletionInput) (*ct.CategoryDeletionPreview, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.CategoryDeletionPreview
	if err := workflow.ExecuteActivity(ctx, ca.PreviewCategoryDeletion, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("preview category deletion: %w", err)
	}
	return &result, nil
}
