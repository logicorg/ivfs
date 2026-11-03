package fx

import (
	"fmt"

	ledgerfx "github.com/logicorg/ivfs/finance/activities/ledger/fx"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// fa is a (nil) pointer to the fx activities struct. It is used to call the activities in the workflow.
var fa = (*ledgerfx.Activities)(nil)

// RegisterFXConversion is a workflow that records a foreign exchange
// conversion as one atomic, idempotent journal entry.
func RegisterFXConversion(ctx workflow.Context, input lt.RegisterFXConversionInput) (*lt.RegisterFXConversionResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())
	input.IdempotencyKey = workflows.DefaultIdempotencyKey(ctx, input.IdempotencyKey)

	var result lt.RegisterFXConversionResult
	if err := workflow.ExecuteActivity(ctx, fa.RegisterFXConversion, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("register fx conversion: %w", err)
	}
	return &result, nil
}
