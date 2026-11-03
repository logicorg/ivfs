package currencies

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateCurrency(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_currency",
		Description: "Create a currency, or return the existing one with the same code, compared case-insensitively. Every " +
			"ledger account and category has a currency, so create the ones you need before setting up accounts; call " +
			"list_currencies first to see which already exist. Safe to call again with the same code.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, ct.InsertCurrencyParams) (ct.InsertCurrencyResult, error),
		ct.InsertCurrencyParams, ct.InsertCurrencyResult,
	](c, currencieswf.InsertCurrency))
}
