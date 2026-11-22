package fx

import (
	"fmt"

	ledgerfx "github.com/logicorg/ivfs/finance/activities/ledger/fx"
	fxt "github.com/logicorg/ivfs/finance/common/types/fxtypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// fa is a (nil) pointer to the fx activities struct. It is used to call the activities in the workflow.
var fa = (*ledgerfx.Activities)(nil)

// RegisterFXConversion is a workflow that records a foreign exchange
// conversion as one atomic, idempotent journal entry. A fee names its
// category, and is debited on the "Expenses" account of the source currency,
// which the workflow finds.
func RegisterFXConversion(ctx workflow.Context, input fxt.RegisterFXConversionInput) (*fxt.RegisterFXConversionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)
	if input.Fee != nil {
		fee, err := feeOnExpenses(ctx, input)
		if err != nil {
			return nil, err
		}
		input.Fee = fee
	}

	var result fxt.RegisterFXConversionResult
	if err := workflow.ExecuteActivity(ctx, fa.RegisterFXConversion, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register fx conversion: %w", err)
	}
	return &result, nil
}

// feeOnExpenses returns the conversion's fee with the "Expenses" account of
// the source currency it is debited on, which its category must be an expense
// one to be booked on.
func feeOnExpenses(ctx workflow.Context, input fxt.RegisterFXConversionInput) (*fxt.FXFee, error) {
	fee := *input.Fee
	if fee.CategoryID <= 0 {
		return nil, internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Fee.CategoryID"}}
	}

	var source lt.RetrieveLedgerAccountResult
	err := workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: input.SourceLedgerAccountID}).Get(ctx, &source)
	if err != nil {
		return nil, fmt.Errorf("retrieve ledger account %d: %w", input.SourceLedgerAccountID, err)
	}

	fee.LedgerAccountID, err = categorieswf.CategoryAccount(ctx, fee.CategoryID, model.LedgerAccountExpense, source.LedgerAccount.CurrencyID)
	if err != nil {
		return nil, err
	}
	return &fee, nil
}
