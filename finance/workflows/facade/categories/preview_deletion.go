package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"go.temporal.io/sdk/workflow"
)

// PreviewCategoryDeletion reports what DeleteCategory would do, and changes
// nothing. It refuses an account that is not income or expense, as the
// deletion does, and a category that does not exist is ErrNotFound.
func PreviewCategoryDeletion(ctx workflow.Context, input ct.PreviewCategoryDeletionInput) (*lt.AccountDeletionPreview, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var retrieved lt.RetrieveLedgerAccountResult
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: input.LedgerAccountID}).Get(ctx, &retrieved); err != nil {
		return nil, fmt.Errorf("retrieve category: %w", err)
	}
	if err := refuseNonCategory(retrieved.LedgerAccount); err != nil {
		return nil, err
	}

	var preview lt.AccountDeletionPreview
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.PreviewAccountDeletion,
		lt.PreviewAccountDeletionInput{LedgerAccountID: &input.LedgerAccountID}).Get(ctx, &preview); err != nil {
		return nil, fmt.Errorf("preview category deletion: %w", err)
	}
	return &preview, nil
}
