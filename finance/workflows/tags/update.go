package tags

import (
	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateTag updates a tag
func UpdateTag(ctx workflow.Context, params *tagtypes.UpdateTagParams) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	a := tag.Tag{}
	err := workflow.ExecuteActivity(ctx, a.UpdateTag, *params).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
