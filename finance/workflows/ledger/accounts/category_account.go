package accounts

import (
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// EnsureCategoryAccount returns the id of the "Expenses" or "Income" account
// of a currency, created the first time the currency needs it. It is not a
// workflow of its own: the workflow that books a categorized posting calls it,
// and it runs CreateLedgerAccount as that workflow's child.
func EnsureCategoryAccount(ctx workflow.Context, kind model.LedgerAccountKind, currencyID int64) (int64, error) {
	name := lt.CategoryAccountName(kind)
	var created lt.CreateLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, CreateLedgerAccount, lt.CreateLedgerAccountInput{
		CurrencyID: currencyID, Name: name, Kind: kind,
	}).Get(ctx, &created)
	if err != nil {
		return 0, fmt.Errorf("find the %s account of currency %d: %w", name, currencyID, err)
	}
	return created.LedgerAccountID, nil
}
