package categories

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every categories-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerListCategories(server, c)
	registerCreateCategory(server, c)
}
