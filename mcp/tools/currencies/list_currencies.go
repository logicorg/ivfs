package currencies

import (
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	currencieswf "github.com/logicorg/ivfs/finance/workflows/currencies"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerListCurrencies(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_currencies",
		Description: "List every currency with its id, ISO code, name, symbol, and precision (decimal places), to resolve a " +
			"reference like \"EUR\" or \"euro\" to a currency id before creating an account or recording a cost basis.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, ct.ListCurrenciesParams) (*ct.ListCurrenciesResult, error),
		ct.ListCurrenciesParams, ct.ListCurrenciesResult,
	](c, currencieswf.ListCurrencies))
}
