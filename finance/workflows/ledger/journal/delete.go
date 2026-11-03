package journal

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteJournalEntry deletes a journal entry with its postings, movements, and
// reversals in one atomic activity. The activity is idempotent, so a retry needs
// no key.
func DeleteJournalEntry(ctx workflow.Context, input lt.DeleteJournalEntryInput) (*lt.DeleteJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.DeleteJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.DeleteJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("delete journal entry: %w", err)
	}

	return &result, nil
}
