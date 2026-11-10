package journal

import (
	"context"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	"github.com/logicorg/ivfs/internal/pkg/workflowrun"
	"github.com/logicorg/ivfs/mcp/mcptypes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"
)

func registerListJournalEntries(server *mcp.Server, c client.Client) {
	mcptypes.AddTool(server, &mcp.Tool{
		Name: "list_journal_entries",
		Description: "Page through posted journal entries, filtered by ledger account, counterparty, exact external id, status, and a " +
			"date range. Pass the previous call's Next as After to fetch the following page.",
	}, listJournalEntriesHandler(c))
}

// listJournalEntriesHandler does not use mcptypes.WorkflowWithResult:
// ListJournalEntriesResult carries model.JournalEntry values, whose relation
// fields form a cycle unusable as an MCP output schema (see
// mcptypes.WorkflowWithResult's doc comment), so this maps the result to
// JournalEntrySummary by hand instead.
func listJournalEntriesHandler(c client.Client) func(
	ctx context.Context, req *mcp.CallToolRequest, args lt.ListJournalEntriesParams,
) (*mcp.CallToolResult, *ListJournalEntriesToolResult, error) {
	return func(
		ctx context.Context, req *mcp.CallToolRequest, args lt.ListJournalEntriesParams,
	) (*mcp.CallToolResult, *ListJournalEntriesToolResult, error) {
		result, err := workflowrun.Run[lt.ListJournalEntriesResult](ctx, c, journalwf.ListJournalEntries, args)
		if err != nil {
			return nil, nil, err
		}

		out := &ListJournalEntriesToolResult{Entries: make([]JournalEntrySummary, len(result.Entries)), Next: result.Next}
		for i, je := range result.Entries {
			postings := make([]PostingSummary, len(je.Postings))
			for j, p := range je.Postings {
				postings[j] = PostingSummary{LedgerAccountID: p.LedgerAccountID, LineNumber: p.LineNumber, Amount: p.Amount, Side: string(p.Side)}
			}
			out.Entries[i] = JournalEntrySummary{
				ID: je.ID, Status: string(je.Status), OccurredAt: je.OccurredAt,
				Description: je.Description, CounterpartyID: je.CounterpartyID,
				ExternalID: je.ExternalID,
				Postings:   postings,
			}
		}
		return nil, out, nil
	}
}
