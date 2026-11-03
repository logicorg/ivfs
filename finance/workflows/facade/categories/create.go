package categories

import (
	"fmt"
	"slices"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

var categoryLedgerAccountKinds = []model.LedgerAccountKind{model.LedgerAccountIncome, model.LedgerAccountExpense}

// CreateCategory creates a new category (ledger account) with the specified input parameters.
func CreateCategory(ctx workflow.Context, input ct.CreateCategoryInput) (*ct.CategoryResult, error) {
	if !slices.Contains(categoryLedgerAccountKinds, input.Kind) {
		return nil, internalerrors.ErrInvalidParameter{Description: `category kind must be "income" or "expense"`}
	}

	var result lt.CreateLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		ParentID: input.ParentID, CurrencyID: input.CurrencyID, Name: input.Name, Kind: input.Kind,
	}).Get(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("create category: %w", err)
	}

	return &ct.CategoryResult{LedgerAccountID: result.LedgerAccountID}, nil
}
