package assetvalues

import (
	"github.com/logicorg/ivfs/finance/activities/asset/value"
	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RetrieveAssetValue retrieves a asset value record.
func RetrieveAssetValue(ctx workflow.Context, params avt.RetrieveAssetValueParams) (*avt.RetrieveAssetValueResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var ava *value.AssetValueActivity

	res := new(avt.RetrieveAssetValueResult)
	err := workflow.ExecuteActivity(ctx, ava.RetrieveAssetValue, params).Get(ctx, res)
	return res, err
}