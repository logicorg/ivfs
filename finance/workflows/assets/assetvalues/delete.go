// Package assetvalues contains the workflows to handle asset values fluctuacitons over time
package assetvalues

import (
	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteAssetValue deletes an assetvalue.
func DeleteAssetValue(ctx workflow.Context, params avt.DeleteAssetValueParams) (*avt.DeleteAssetValueResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var ava *value.AssetValueActivity

	res := new(avt.DeleteAssetValueResult)
	err := workflow.ExecuteActivity(ctx, ava.DeleteAssetValue, params).Get(ctx, res)
	return res, err
}
