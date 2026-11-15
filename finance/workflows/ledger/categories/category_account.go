package categories

import (
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// CategoryAccount returns the id of the account a posting under the category
// is booked on: the "Expenses" or "Income" account of the currency, created
// the first time the currency needs it. It fails unless the category is of
// the kind, which would otherwise put an expense category on "Income" or the
// reverse. A category has no currency and may be closed: closing one only
// stops it being offered.
//
// It is not a workflow of its own: the workflow that books the posting calls
// it, and it runs RetrieveCategory and CreateLedgerAccount as that workflow's
// children.
func CategoryAccount(ctx workflow.Context, categoryID int64, kind model.LedgerAccountKind, currencyID int64) (int64, error) {
	var retrieved ct.RetrieveCategoryResult
	if err := workflow.ExecuteChildWorkflow(ctx, RetrieveCategory, ct.RetrieveCategoryParams{ID: categoryID}).Get(ctx, &retrieved); err != nil {
		return 0, fmt.Errorf("retrieve category %d: %w", categoryID, err)
	}
	if err := ct.CheckKind(retrieved.Category, kind); err != nil {
		return 0, err
	}

	name := lt.CategoryAccountName(kind)
	var created lt.CreateLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.CreateLedgerAccount, lt.CreateLedgerAccountInput{
		CurrencyID: currencyID, Name: name, Kind: kind,
	}).Get(ctx, &created)
	if err != nil {
		return 0, fmt.Errorf("find the %s account of currency %d: %w", name, currencyID, err)
	}
	return created.LedgerAccountID, nil
}
