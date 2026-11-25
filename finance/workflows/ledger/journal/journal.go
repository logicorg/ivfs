package journal

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

var (
	ledgerActivity = (*ledgeractivity.Activities)(nil)
)

func PostJournalEntry(ctx workflow.Context, input jt.PostJournalEntryInput) (*jt.PostJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)

	var result jt.PostJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.PostJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("post journal entry: %w", err)
	}

	return &result, nil
}

func RetrieveJournalEntry(ctx workflow.Context, params jt.RetrieveJournalEntryParams) (*jt.RetrieveJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result jt.RetrieveJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RetrieveJournalEntry, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve journal entry: %w", err)
	}

	return &result, nil
}
