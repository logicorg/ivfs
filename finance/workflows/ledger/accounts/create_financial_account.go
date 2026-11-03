package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// CreateFinancialAccount is a workflow that creates a financial account. It
// refuses to create "Untracked accounts" under the "Untracked" institution,
// which only the ledger creates.
func CreateFinancialAccount(ctx workflow.Context, input lt.CreateFinancialAccountInput) (*lt.CreateFinancialAccountResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	untracked, err := isUntrackedFinancialAccount(ctx, input.InstitutionID, input.Name)
	if err != nil {
		return nil, fmt.Errorf("create financial account: %w", err)
	}
	if untracked {
		return nil, becomingUntrackedFinancialAccountError(input.Name, "created")
	}

	var result lt.CreateFinancialAccountResult
	if err := workflow.ExecuteActivity(ctx, la.CreateFinancialAccount, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("create financial account: %w", err)
	}
	return &result, nil
}
