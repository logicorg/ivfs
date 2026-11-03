package assets

import (
	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateAsset updates an asset
func UpdateAsset(ctx workflow.Context, params at.UpdateAssetParams) (*at.UpdateAssetResult, error) {
	ao := workflows.GetDefaultActivityOptions()
	ctx = workflow.WithActivityOptions(ctx, ao)

	var a *asset.AssetActivity
	var result at.UpdateAssetResult
	err := workflow.ExecuteActivity(ctx, a.UpdateAsset, params).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
