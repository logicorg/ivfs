package accounts

import (
	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	counterpartieswf "github.com/logicorg/ivfs/finance/workflows/ledger/counterparties"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerListCounterparties(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name:        "list_counterparties",
		Description: "Search counterparties (payees/payers referenced by journal entries) by a substring of their name.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, cpt.ListCounterpartiesParams) (*cpt.ListCounterpartiesResult, error),
		cpt.ListCounterpartiesParams, cpt.ListCounterpartiesResult,
	](c, counterpartieswf.ListCounterparties))
}
