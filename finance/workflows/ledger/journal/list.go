package journal

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

func ListJournalEntries(ctx workflow.Context, params lt.ListJournalEntriesParams) (*lt.ListJournalEntriesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.ListJournalEntriesResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ListJournalEntries, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	return &result, nil
}
