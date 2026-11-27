package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// NetWorthSeries returns the balances and the units behind net worth over a
// range of days, for the UI to value (ADR 0015).
func NetWorthSeries(ctx workflow.Context, params rpt.NetWorthSeriesParams) (*rpt.NetWorthSeriesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result rpt.NetWorthSeriesResult
	if err := workflow.ExecuteActivity(ctx, la.NetWorthSeries, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("net worth series: %w", err)
	}
	return &result, nil
}
