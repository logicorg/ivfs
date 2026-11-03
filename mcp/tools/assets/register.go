package assets

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every assets-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerListAssets(server, c)
}
