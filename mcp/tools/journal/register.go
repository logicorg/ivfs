package journal

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

// Register adds every journal-domain MCP tool to server.
func Register(server *mcp.Server, c client.Client) {
	registerPostJournalEntry(server, c)
	registerListJournalEntries(server, c)
}
