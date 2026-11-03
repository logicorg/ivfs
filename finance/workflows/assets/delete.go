package assets

import (
	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteAssset deletes an asset
func DeleteAsset(ctx workflow.Context, params at.DeleteAssetParams) (*at.DeleteAssetResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var aa *asset.AssetActivity

	var result at.DeleteAssetResult
	err := workflow.ExecuteActivity(ctx, aa.DeleteAsset, params).Get(ctx, &result)
	return &result, err
}
