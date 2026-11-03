package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CloseFinancialAccount is a workflow that closes a financial account. It
// refuses to close the stray financial account, since every later deletion
// that moves units onto it would fail on the closed account.
func CloseFinancialAccount(ctx workflow.Context, input lt.CloseFinancialAccountInput) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	account, err := retrieveFinancialAccount(ctx, input.FinancialAccountID)
	if err != nil {
		return fmt.Errorf("close financial account: %w", err)
	}
	untracked, err := isUntrackedFinancialAccount(ctx, account.InstitutionID, account.Name)
	if err != nil {
		return fmt.Errorf("close financial account: %w", err)
	}
	if untracked {
		return untrackedFinancialAccountError(account.Name, "closed")
	}

	if err := workflow.ExecuteActivity(ctx, la.CloseFinancialAccount, input).Get(ctx, nil); err != nil {
		return fmt.Errorf("close financial account: %w", err)
	}
	return nil
}
