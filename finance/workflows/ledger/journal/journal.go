package journal

import (
	"fmt"

	ledgeractivity "github.com/logicorg/ivfs/finance/activities/ledger/journal"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

var (
	ledgerActivity = (*ledgeractivity.Activities)(nil)
)

func PostJournalEntry(ctx workflow.Context, input lt.PostJournalEntryInput) (*lt.PostJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)

	var result lt.PostJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.PostJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("post journal entry: %w", err)
	}

	return &result, nil
}

func RetrieveJournalEntry(ctx workflow.Context, params lt.RetrieveJournalEntryParams) (*lt.RetrieveJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result lt.RetrieveJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.RetrieveJournalEntry, params).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve journal entry: %w", err)
	}

	return &result, nil
}

func ReverseJournalEntry(ctx workflow.Context, input lt.ReverseJournalEntryInput) (*lt.ReverseJournalEntryResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)

	var result lt.ReverseJournalEntryResult
	if err := workflow.ExecuteActivity(ctx, ledgerActivity.ReverseJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("reverse journal entry: %w", err)
	}

	return &result, nil
}
