package journal

import (
	"fmt"

	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// SetJournalEntryTags replaces the tags of an entry in one activity. The
// activity is idempotent, so a retry needs no key.
func SetJournalEntryTags(ctx workflow.Context, input jt.SetJournalEntryTagsInput) (*jt.SetJournalEntryTagsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result jt.SetJournalEntryTagsResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.SetJournalEntryTags, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("set journal entry tags: %w", err)
	}

	return &result, nil
}
