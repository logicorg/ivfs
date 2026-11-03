package institution

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every institution-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerCreateInstitution(server, c)
}
