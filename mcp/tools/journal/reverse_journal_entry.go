package journal

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/workflow"
)

func registerReverseJournalEntry(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "reverse_journal_entry",
		Description: "Reverse a posted journal entry with a new, balanced reversing entry. The original entry is never changed or " +
			"deleted; this is how a mistake is corrected.",
	}, mcptypes.WorkflowWithResult[
		func(workflow.Context, lt.ReverseJournalEntryInput) (*lt.ReverseJournalEntryResult, error),
		ReverseJournalEntryToolInput, lt.ReverseJournalEntryResult,
	](c, journalwf.ReverseJournalEntry))
}
