package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
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
		func(workflow.Context, lt.ListCounterpartiesParams) (*lt.ListCounterpartiesResult, error),
		lt.ListCounterpartiesParams, lt.ListCounterpartiesResult,
	](c, ledgeraccountswf.ListCounterparties))
}
