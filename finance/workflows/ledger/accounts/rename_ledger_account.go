package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RenameLedgerAccount renames a ledger account. It refuses to rename a shared
// or stray account, or to give a top-level account one of their names.
func RenameLedgerAccount(ctx workflow.Context, input lt.RenameLedgerAccountInput) (*lt.LedgerAccountMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	account, err := retrieveLedgerAccount(ctx, input.LedgerAccountID)
	if err != nil {
		return nil, fmt.Errorf("rename ledger account: %w", err)
	}
	if err := refuseReservedAccount(account, "renamed"); err != nil {
		return nil, err
	}
	renamed := *account
	renamed.Name = input.Name
	if err := refuseBecomingReserved(account, &renamed, "renamed"); err != nil {
		return nil, err
	}

	var result lt.LedgerAccountMutationResult
	if err := workflow.ExecuteActivity(ctx, la.RenameLedgerAccount, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("rename ledger account: %w", err)
	}
	return &result, nil
}
