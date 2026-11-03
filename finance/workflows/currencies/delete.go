package currencies

import (
	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteCurrency deletes a currency using a Temporal workflow.
func DeleteCurrency(ctx workflow.Context, params *ct.DeleteCurrencyParams) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	ca := currency.Currency{}
	err := workflow.ExecuteActivity(ctx, ca.DeleteCurrency, params).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
