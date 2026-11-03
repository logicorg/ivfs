package tags

import (
	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// DeleteTag deletes a tag
func DeleteTag(ctx workflow.Context, params *tagtypes.DeleteTagParams) error {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	activityParams := tagtypes.DeleteTagParams{
		ID:   params.ID,
		Name: params.Name,
	}

	a := tag.Tag{}
	err := workflow.ExecuteActivity(ctx, a.DeleteTag, activityParams).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
