package currencies

import (
	"fmt"

	"github.com/logicorg/ivfs/finance/activities/currency/currency"
	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RetrieveCurrency retrieves a currency using a Temporal workflow.
func RetrieveCurrency(ctx workflow.Context, params *ct.RetrieveCurrencyParams) (*model.Currency, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	if err := params.Validate(); err != nil {
		return nil, err
	}

	ca := (*currency.Currency)(nil)

	retrieved := new(model.Currency)
	if params.ID > 0 {
		err := workflow.ExecuteActivity(ctx, ca.RetrieveCurrencyByID, params.ID).Get(ctx, retrieved)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve currency by ID: %w", err)
		}
		return retrieved, nil
	}

	err := workflow.ExecuteActivity(ctx, ca.FindCurrencyBy, ct.FindCurrencyByParams{
		Name:   params.Name,
		Code:   params.Code,
		Symbol: params.Symbol,
	}).Get(ctx, retrieved)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve currency: %w", err)
	}

	return retrieved, nil
}
