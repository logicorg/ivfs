package portfolio

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	portfoliowf "github.com/logicorg/ivfs/finance/workflows/ledger/portfolio"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerRegisterVest(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "register_vest",
		Description: "Record the units received from an RSU vest together with the entry that books the compensation - income and any " +
			"tax withheld in shares or cash. Sell-to-cover and an ESPP purchase are ordinary register_investment_trade calls, not this tool. " +
			"Look the ledger and financial account ids up first with list_accounts, and asset ids with list_assets, rather than guessing them.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.RegisterVestInput) (*lt.RegisterVestResult, error),
		RegisterVestToolInput, lt.RegisterVestResult,
	](c, portfoliowf.RegisterVest))
}
