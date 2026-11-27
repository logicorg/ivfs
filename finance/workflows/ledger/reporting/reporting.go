package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	rpt "github.com/logicorg/ivfs/finance/common/types/reportingtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func LedgerAccountBalance(ctx workflow.Context, params jt.LedgerAccountBalanceParams) (*jt.LedgerAccountBalanceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result jt.LedgerAccountBalanceResult
	if err := workflow.ExecuteActivity(ctx, la.LedgerAccountBalance, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate ledger account balance: %w", err)
	}
	return &result, nil
}

func FinancialAccountBalances(ctx workflow.Context, params rpt.FinancialAccountBalancesParams) (*rpt.FinancialAccountBalancesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result rpt.FinancialAccountBalancesResult
	if err := workflow.ExecuteActivity(ctx, la.FinancialAccountBalances, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve financial account balances: %w", err)
	}
	return &result, nil
}

func ListPositionsAt(ctx workflow.Context, params rpt.PositionsAtParams) (*rpt.PositionsAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result rpt.PositionsAtResult
	if err := workflow.ExecuteActivity(ctx, la.PositionsAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve positions: %w", err)
	}
	return &result, nil
}

func ListPostings(ctx workflow.Context, params jt.ListPostingsParams) (*jt.ListPostingsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result jt.ListPostingsResult
	if err := workflow.ExecuteActivity(ctx, (&ledgeractivity.Activities{}).ListPostings, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list postings: %w", err)
	}
	return &result, nil
}

func TrialBalanceAt(ctx workflow.Context, params jt.TrialBalanceAtParams) (*jt.TrialBalanceAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result jt.TrialBalanceAtResult
	if err := workflow.ExecuteActivity(ctx, (&ledgeractivity.Activities{}).TrialBalanceAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate trial balance: %w", err)
	}
	return &result, nil
}
