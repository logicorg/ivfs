package journal

import (
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
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
		func(workflow.Context, jt.PostJournalEntryInput) (*jt.PostJournalEntryResult, error),
		PostJournalEntryToolInput, jt.PostJournalEntryResult,
	](c, journalwf.PostJournalEntry))
}
