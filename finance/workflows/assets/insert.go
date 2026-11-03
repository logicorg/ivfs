package assets

import (
	"time"

	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"go.temporal.io/sdk/workflow"
)

// InsertAsset inserts an asset in the DB
func InsertAsset(ctx workflow.Context, params at.InsertAssetParams) (*at.InsertAssetResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{StartToCloseTimeout: 10 * time.Second})

	aa := asset.AssetActivity{}

	result := new(at.InsertAssetResult)
	err := workflow.ExecuteActivity(ctx, aa.InsertAsset, params).Get(ctx, result)
	return result, err
}
