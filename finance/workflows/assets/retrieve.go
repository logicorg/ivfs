package assets

import (
	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RetrieveAsset retrieves an asset
func RetrieveAsset(ctx workflow.Context, params *at.RetrieveAssetParams) (*at.RetrieveAssetResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	var result *at.RetrieveAssetResult
	a := asset.AssetActivity{}
	err := workflow.ExecuteActivity(ctx, a.RetrieveAsset, params).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return result, err
}

// RetrieveAllAssets retrieves all assets in the DB
func RetrieveAllAssets(ctx workflow.Context) (*at.RetrieveAllAssetsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	var activityResult []*model.Asset
	a := asset.AssetActivity{}
	err := workflow.ExecuteActivity(ctx, a.RetrieveAllAssets).Get(ctx, &activityResult)
	if err != nil {
		return nil, err
	}

	result := &at.RetrieveAllAssetsResult{
		Assets: activityResult,
	}

	return result, nil
}
