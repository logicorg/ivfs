package accounts

import (
	cpt "github.com/logicorg/ivfs/finance/common/types/counterpartytypes"
	counterpartieswf "github.com/logicorg/ivfs/finance/workflows/ledger/counterparties"
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
		func(workflow.Context, cpt.CreateCounterpartyInput) (*cpt.CreateCounterpartyResult, error),
		cpt.CreateCounterpartyInput, cpt.CreateCounterpartyResult,
	](c, counterpartieswf.CreateCounterparty))
}
