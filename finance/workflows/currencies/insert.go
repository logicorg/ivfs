package currencies

import (
	"time"

	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"go.temporal.io/sdk/workflow"
)

// InsertCurrency inserts a currency in the database
func InsertCurrency(ctx workflow.Context, params ct.InsertCurrencyParams) (ct.InsertCurrencyResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second})

	currencyActivity := &currency.Currency{}

	res := ct.InsertCurrencyResult{}
	err := workflow.ExecuteActivity(ctx, currencyActivity.InsertCurrency, params).Get(ctx, &res)
	return res, err
}
