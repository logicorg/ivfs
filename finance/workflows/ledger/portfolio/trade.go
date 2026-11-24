package portfolio

import (
	"fmt"

	pft "github.com/logicorg/ivfs/finance/common/types/portfoliotypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func RegisterInvestmentTrade(ctx workflow.Context, input pft.RegisterInvestmentTradeInput) (*pft.RegisterInvestmentTradeResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.Entry.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.Entry.IdempotencyKey)
	var result pft.RegisterInvestmentTradeResult

	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RegisterInvestmentTrade, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register investment trade: %w", err)
	}

	return &result, nil
}
