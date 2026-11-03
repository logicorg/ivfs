package assets

import (
	"github.com/logicorg/ivfs/finance/activities/asset"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// ListAssets is a workflow that searches assets by name and/or ticker
// substring.
func ListAssets(ctx workflow.Context, params at.ListAssetsParams) (*at.ListAssetsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var result at.ListAssetsResult
	a := asset.AssetActivity{}
	if err := workflow.ExecuteActivity(ctx, a.ListAssets, params).Get(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
