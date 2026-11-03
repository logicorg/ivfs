package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CategoryTotals totals each income and expense category over a period.
func CategoryTotals(ctx workflow.Context, params lt.CategoryTotalsParams) (*lt.CategoryTotalsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result lt.CategoryTotalsResult
	if err := workflow.ExecuteActivity(ctx, la.CategoryTotals, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("total categories: %w", err)
	}
	return &result, nil
}
