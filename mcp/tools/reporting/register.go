// Package reporting holds the MCP tools that report balances from the ledger.
package reporting

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every reporting-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerGetFinancialAccountBalances(server, c)
	registerGetLedgerAccountBalance(server, c)
}
