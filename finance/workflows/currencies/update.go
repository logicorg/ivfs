package currencies

import (
	"time"

	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"go.temporal.io/sdk/workflow"
)

// UpdateCurrency updates an existing currency using a Temporal workflow.
func UpdateCurrency(ctx workflow.Context, params *ct.UpdateCurrencyParams) error {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second})

	ca := &currency.Currency{}
	err := workflow.ExecuteActivity(ctx, ca.UpdateCurrency, params).Get(ctx, nil)
	return err
}
