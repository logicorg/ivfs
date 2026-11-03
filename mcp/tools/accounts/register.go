package accounts

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every accounts-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerListAccounts(server, c)
	registerListFinancialAccounts(server, c)
	registerListCounterparties(server, c)
	registerCreateFinancialAccount(server, c)
	registerCreateLedgerAccount(server, c)
	registerCreateCounterparty(server, c)
}
