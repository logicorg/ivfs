package journal

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerPostJournalEntry(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "post_journal_entry",
		Description: "Post a balanced double-entry journal entry: two or more postings on existing ledger accounts, " +
			"whose amounts net to zero per currency. Look ledger account ids up first with list_accounts rather than guessing them.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.PostJournalEntryInput) (*lt.PostJournalEntryResult, error),
		PostJournalEntryToolInput, lt.PostJournalEntryResult,
	](c, journalwf.PostJournalEntry))
}
