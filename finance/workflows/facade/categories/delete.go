package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// DeleteCategory deletes an income or expense account, as ADR 0012
// describes: its postings on kept entries move to "Uncategorized" of its kind
// and currency, so totals spent and earned do not change, only their
// breakdown. With a replacement it is a merge instead.
//
// It retrieves the account first and refuses one that is not income or
// expense with ErrInvalidParameter, so the categories endpoint cannot delete a
// balance; a category that does not exist succeeds. It then runs
// DeleteLedgerAccount as a child under DeleteLedgerAccountWorkflowID, through
// ExecuteDeletionChild, so a deletion of the same account started from the
// ledger accounts endpoint is waited for rather than run twice.
func DeleteCategory(ctx workflow.Context, input ct.DeleteCategoryInput) (*lt.DeleteAccountResult, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}

	var retrieved lt.RetrieveLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: input.LedgerAccountID}).Get(ctx, &retrieved)
	if workflows.HasErrorType(err, "ErrNotFound") {
		return &lt.DeleteAccountResult{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("retrieve category: %w", err)
	}
	if err := refuseNonCategory(retrieved.LedgerAccount); err != nil {
		return nil, err
	}

	var result lt.DeleteAccountResult
	if err := workflows.ExecuteDeletionChild(ctx, workflows.DeleteLedgerAccountWorkflowID(input.LedgerAccountID), ledgeraccountswf.DeleteLedgerAccount,
		lt.DeleteLedgerAccountInput{LedgerAccountID: input.LedgerAccountID, ReplacementID: input.ReplacementID}, &result); err != nil {
		return nil, fmt.Errorf("delete category: %w", err)
	}
	return &result, nil
}

// refuseNonCategory refuses a ledger account that is not income or expense,
// so the categories endpoints cannot delete a balance.
func refuseNonCategory(account *model.LedgerAccount) error {
	if account.Kind == model.LedgerAccountIncome || account.Kind == model.LedgerAccountExpense {
		return nil
	}
	return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf(
		"ledger account %d has kind %s, not income or expense; delete it as a ledger account", account.ID, account.Kind)}
}
