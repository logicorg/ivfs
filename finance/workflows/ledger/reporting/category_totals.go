package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CategoryTotals totals each income and expense category, and what has
// none, by currency over a period.
func CategoryTotals(ctx workflow.Context, params rpt.CategoryTotalsParams) (*rpt.CategoryTotalsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result rpt.CategoryTotalsResult
	if err := workflow.ExecuteActivity(ctx, la.CategoryTotals, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("total categories: %w", err)
	}
	return &result, nil
}
