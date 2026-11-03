package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// NetWorthSeries returns the balances and the units behind net worth over a
// range of days, for the UI to value (ADR 0015).
func NetWorthSeries(ctx workflow.Context, params lt.NetWorthSeriesParams) (*lt.NetWorthSeriesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result lt.NetWorthSeriesResult
	if err := workflow.ExecuteActivity(ctx, la.NetWorthSeries, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("net worth series: %w", err)
	}
	return &result, nil
}
