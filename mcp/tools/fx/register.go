package fx

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every fx-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerRegisterFXConversion(server, c)
}
