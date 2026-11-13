package journal

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ReplaceJournalEntry corrects a posted journal entry, replacing its postings
// and movements, in one atomic activity. The activity writes the same state
// when it runs again, so a retry needs no key.
func ReplaceJournalEntry(ctx workflow.Context, input lt.ReplaceJournalEntryInput) (*lt.ReplaceJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ReplaceJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ReplaceJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("replace journal entry: %w", err)
	}

	return &result, nil
}
