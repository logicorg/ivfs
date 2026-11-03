package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CloseLedgerAccount closes a ledger account. It refuses to close a shared or
// stray account, since every later write that needs it would fail on it.
func CloseLedgerAccount(ctx workflow.Context, input lt.CloseLedgerAccountInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	account, err := retrieveLedgerAccount(ctx, input.LedgerAccountID)
	if err != nil {
		return fmt.Errorf("close ledger account: %w", err)
	}
	if err := refuseReservedAccount(account, "closed"); err != nil {
		return err
	}

	if err := workflow.ExecuteActivity(ctx, la.CloseLedgerAccount, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("close ledger account: %w", err)
	}
	return nil
}
