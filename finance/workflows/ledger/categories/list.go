package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ListCategories lists the categories that match the filters.
func ListCategories(ctx workflow.Context, input ct.ListCategoriesParams) (*ct.ListCategoriesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.ListCategoriesResult
	if err := workflow.ExecuteActivity(ctx, ca.ListCategories, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return &result, nil
}
