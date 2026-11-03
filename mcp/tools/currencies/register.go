package currencies

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every currencies-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerCreateCurrency(server, c)
	registerListCurrencies(server, c)
}
