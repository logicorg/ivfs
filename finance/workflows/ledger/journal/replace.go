package journal

import (
	"fmt"

	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ReplaceJournalEntry corrects a posted journal entry, replacing its postings
// and movements, in one atomic activity. The activity writes the same state
// when it runs again, so a retry needs no key.
func ReplaceJournalEntry(ctx workflow.Context, input jt.ReplaceJournalEntryInput) (*jt.ReplaceJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result jt.ReplaceJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ReplaceJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("replace journal entry: %w", err)
	}

	return &result, nil
}
