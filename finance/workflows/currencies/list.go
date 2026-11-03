package currencies

import (
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// ListCurrencies is a workflow that lists every currency.
func ListCurrencies(ctx workflow.Context, _ ct.ListCurrenciesParams) (*ct.ListCurrenciesResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	ca := (*currency.Currency)(nil)

	var currencies []*model.Currency
	if err := workflow.ExecuteActivity(ctx, ca.ListCurrencies).Get(ctx, &currencies); err != nil {
		return nil, fmt.Errorf("list currencies: %w", err)
	}
	return &ct.ListCurrenciesResult{Currencies: currencies}, nil
}
