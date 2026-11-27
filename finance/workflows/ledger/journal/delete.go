package journal

import (
	"fmt"

	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteJournalEntry deletes a journal entry with its postings and movements in
// one atomic activity. The activity is idempotent, so a retry needs no key.
func DeleteJournalEntry(ctx workflow.Context, input jt.DeleteJournalEntryInput) (*jt.DeleteJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result jt.DeleteJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.DeleteJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("delete journal entry: %w", err)
	}

	return &result, nil
}
