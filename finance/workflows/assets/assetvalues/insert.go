package assetvalues

import (
	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// InsertAssetValue creates a new asset value record.
func InsertAssetValue(ctx workflow.Context, params avt.InsertAssetValueParams) (*avt.InsertAssetValueResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	if err := params.Validate(); err != nil {
		return nil, err
	}

	var ava *value.AssetValueActivity

	res := new(avt.InsertAssetValueResult)

	err := workflow.ExecuteActivity(ctx, ava.InsertAssetValue, params).Get(ctx, res)
	return res, err
}
