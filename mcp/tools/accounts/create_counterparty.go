package accounts

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerCreateCounterparty(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "create_counterparty",
		Description: "Create a counterparty (a merchant, employer, or person a journal entry can name), or return the existing one " +
			"with the same name, compared case-insensitively. Safe to call again for the same counterparty.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.CreateCounterpartyInput) (*lt.CreateCounterpartyResult, error),
		lt.CreateCounterpartyInput, lt.CreateCounterpartyResult,
	](c, ledgeraccountswf.CreateCounterparty))
}
