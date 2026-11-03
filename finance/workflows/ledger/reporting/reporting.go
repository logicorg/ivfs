package reporting

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/reporting"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func LedgerAccountBalance(ctx workflow.Context, params lt.LedgerAccountBalanceParams) (*lt.LedgerAccountBalanceResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result lt.LedgerAccountBalanceResult
	if err := workflow.ExecuteActivity(ctx, la.LedgerAccountBalance, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate ledger account balance: %w", err)
	}
	return &result, nil
}

func FinancialAccountBalances(ctx workflow.Context, params lt.FinancialAccountBalancesParams) (*lt.FinancialAccountBalancesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result lt.FinancialAccountBalancesResult
	if err := workflow.ExecuteActivity(ctx, la.FinancialAccountBalances, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve financial account balances: %w", err)
	}
	return &result, nil
}

func ListPositionsAt(ctx workflow.Context, params lt.PositionsAtParams) (*lt.PositionsAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	la := (*ledgeractivity.Activities)(nil)

	var result lt.PositionsAtResult
	if err := workflow.ExecuteActivity(ctx, la.PositionsAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve positions: %w", err)
	}
	return &result, nil
}

func ListPostings(ctx workflow.Context, params lt.ListPostingsParams) (*lt.ListPostingsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result lt.ListPostingsResult
	if err := workflow.ExecuteActivity(ctx, (&ledgeractivity.Activities{}).ListPostings, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list postings: %w", err)
	}
	return &result, nil
}

func TrialBalanceAt(ctx workflow.Context, params lt.TrialBalanceAtParams) (*lt.TrialBalanceAtResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	var result lt.TrialBalanceAtResult
	if err := workflow.ExecuteActivity(ctx, (&ledgeractivity.Activities{}).TrialBalanceAt, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("calculate trial balance: %w", err)
	}
	return &result, nil
}
