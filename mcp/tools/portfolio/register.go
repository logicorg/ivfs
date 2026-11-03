package portfolio

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every portfolio-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerRegisterInvestmentTrade(server, c)
	registerRegisterVest(server, c)
}
