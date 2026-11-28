package journal

import (
	"fmt"

	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func ListJournalEntries(ctx workflow.Context, params jt.ListJournalEntriesParams) (*jt.ListJournalEntriesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result jt.ListJournalEntriesResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ListJournalEntries, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	return &result, nil
}
