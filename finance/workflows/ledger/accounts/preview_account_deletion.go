package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// PreviewAccountDeletion reports what deleting a ledger account, a financial
// account, or an institution would do, and changes nothing (ADR 0012). It
// describes the account as it is when asked. A target that does not exist is
// ErrNotFound, unlike its deletion, since there is nothing to show.
func PreviewAccountDeletion(ctx workflow.Context, input lt.PreviewAccountDeletionInput) (*lt.AccountDeletionPreview, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var preview lt.AccountDeletionPreview
	if err := workflow.ExecuteActivity(ctx, la.PreviewAccountDeletion, input).Get(ctx, &preview); err != nil {
		return nil, fmt.Errorf("preview account deletion: %w", err)
	}
	return &preview, nil
}
