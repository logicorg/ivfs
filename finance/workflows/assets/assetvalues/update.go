package assetvalues

import (
	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateAssetValue updates a asset value record.
func UpdateAssetValue(ctx workflow.Context, params avt.UpdateAssetValueParams) (*avt.UpdateAssetValueResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var ava *value.AssetValueActivity

	res := new(avt.UpdateAssetValueResult)
	err := workflow.ExecuteActivity(ctx, ava.UpdateAssetValue, params).Get(ctx, res)
	return res, err
}