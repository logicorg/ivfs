package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RetrieveCategory retrieves a category.
func RetrieveCategory(ctx workflow.Context, input ct.RetrieveCategoryParams) (*ct.RetrieveCategoryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result ct.RetrieveCategoryResult
	if err := workflow.ExecuteActivity(ctx, ca.RetrieveCategory, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve category: %w", err)
	}
	return &result, nil
}
