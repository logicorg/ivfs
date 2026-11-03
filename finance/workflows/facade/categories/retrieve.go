package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

func RetrieveCategory(ctx workflow.Context, input ct.RetrieveCategoryParams) (*ct.RetrieveCategoryResult, error) {

	var result lt.RetrieveLedgerAccountResult
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: input.ID}).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("retrieve category: %w", err)
	}
	return &ct.RetrieveCategoryResult{LedgerAccount: result.LedgerAccount}, nil
}

func ListCategories(ctx workflow.Context, input ct.ListCategoriesParams) (*ct.ListCategoriesResult, error) {

	var result lt.ListLedgerAccountsResult
	if err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.ListLedgerAccounts, lt.ListLedgerAccountsParams{
		ParentID: input.ParentID, CurrencyID: input.CurrencyID, NameContains: input.NameContains, IncludeClosed: input.IncludeClosed,
		Kinds: []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense},
	}).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return &ct.ListCategoriesResult{LedgerAccounts: result.LedgerAccounts}, nil
}
