package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"go.temporal.io/sdk/workflow"
)

func CloseCategory(ctx workflow.Context, input ct.CloseCategoryInput) error {
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CloseLedgerAccount, lt.CloseLedgerAccountInput{
		LedgerAccountID: input.LedgerAccountID, ClosedAt: input.ClosedAt,
	}).Get(ctx, nil); err != nil {
		return fmt.Errorf("close category: %w", err)
	}
	return nil
}
