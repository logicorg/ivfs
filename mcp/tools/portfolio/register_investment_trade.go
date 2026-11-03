package portfolio

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerRegisterInvestmentTrade(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "register_investment_trade",
		Description: "Record one buy or sell of a tradable asset and its cash-side journal entry atomically. " +
			"Look the ledger and financial account ids up first with list_accounts, and asset ids with list_assets, rather than guessing them.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.RegisterInvestmentTradeInput) (*lt.RegisterInvestmentTradeResult, error),
		RegisterInvestmentTradeToolInput, lt.RegisterInvestmentTradeResult,
	](c, portfoliowf.RegisterInvestmentTrade))
}
