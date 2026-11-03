package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"go.temporal.io/sdk/workflow"
)

func RenameCategory(ctx workflow.Context, input ct.RenameCategoryInput) (*ct.CategoryResult, error) {

	var result lt.LedgerAccountMutationResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RenameLedgerAccount, lt.RenameLedgerAccountInput{
		LedgerAccountID: input.LedgerAccountID, Name: input.Name,
	}).Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("rename category: %w", err)
	}

	return &ct.CategoryResult{LedgerAccountID: result.LedgerAccountID}, nil
}

func MoveCategory(ctx workflow.Context, input ct.MoveCategoryInput) (*ct.CategoryResult, error) {

	if err := input.Validate(); err != nil {
		return nil, err
	}

	var result lt.LedgerAccountMutationResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.MoveLedgerAccount, lt.MoveLedgerAccountInput{
		LedgerAccountID: input.LedgerAccountID, ParentID: input.ParentID,
	}).Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("move category: %w", err)
	}

	return &ct.CategoryResult{LedgerAccountID: result.LedgerAccountID}, nil
}
