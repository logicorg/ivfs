package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// MoveLedgerAccount moves a ledger account under another parent, or to the top
// level. It refuses to move a shared or stray account, or to move an account
// of one of their names to the top level.
func MoveLedgerAccount(ctx workflow.Context, input lt.MoveLedgerAccountInput) (*lt.LedgerAccountMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	account, err := retrieveLedgerAccount(ctx, input.LedgerAccountID)
	if err != nil {
		return nil, fmt.Errorf("move ledger account: %w", err)
	}
	if err := refuseReservedAccount(account, "moved"); err != nil {
		return nil, err
	}
	moved := *account
	moved.ParentID = input.ParentID
	if err := refuseBecomingReserved(account, &moved, "moved to the top level"); err != nil {
		return nil, err
	}

	var result lt.LedgerAccountMutationResult
	if err := workflow.ExecuteActivity(ctx, la.MoveLedgerAccount, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("move ledger account: %w", err)
	}
	return &result, nil
}
