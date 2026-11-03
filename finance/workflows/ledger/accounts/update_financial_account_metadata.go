package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateFinancialAccountMetadata is a workflow that renames a financial
// account and/or updates its external ID. It refuses to change the stray
// financial account, or to rename another account of the "Untracked"
// institution onto its name.
func UpdateFinancialAccountMetadata(ctx workflow.Context, input lt.UpdateFinancialAccountMetadataInput) (*lt.FinancialAccountMutationResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	account, err := retrieveFinancialAccount(ctx, input.FinancialAccountID)
	if err != nil {
		return nil, fmt.Errorf("update financial account metadata: %w", err)
	}
	untracked, err := isUntrackedFinancialAccount(ctx, account.InstitutionID, account.Name)
	if err != nil {
		return nil, fmt.Errorf("update financial account metadata: %w", err)
	}
	if untracked {
		return nil, untrackedFinancialAccountError(account.Name, "updated")
	}
	if input.Name != nil {
		untracked, err := isUntrackedFinancialAccount(ctx, account.InstitutionID, *input.Name)
		if err != nil {
			return nil, fmt.Errorf("update financial account metadata: %w", err)
		}
		if untracked {
			return nil, becomingUntrackedFinancialAccountError(account.Name, fmt.Sprintf("renamed to %q", *input.Name))
		}
	}

	var result lt.FinancialAccountMutationResult
	if err := workflow.ExecuteActivity(ctx, la.UpdateFinancialAccountMetadata, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("update financial account metadata: %w", err)
	}
	return &result, nil
}
