package tags

import (
	"github.com/logicorg/ivfs/finance/activities/tag"
	"github.com/logicorg/ivfs/finance/common/types/tagtypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// RetrieveTag retrieves a tag
func RetrieveTag(ctx workflow.Context, params *tagtypes.RetrieveTagParams) (*tagtypes.RetrieveTagResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	activityParams := tagtypes.RetrieveTagParams{
		ID:   params.ID,
		Name: params.Name,
	}

	var activityResult *tagtypes.RetrieveTagResult
	a := tag.Tag{}
	err := workflow.ExecuteActivity(ctx, a.RetrieveTag, activityParams).Get(ctx, &activityResult)
	if err != nil {
		return nil, err
	}

	result := &tagtypes.RetrieveTagResult{
		Tag: activityResult.Tag,
	}

	return result, nil
}

// ListTags retrieves all tags in the DB
func ListTags(ctx workflow.Context) (*tagtypes.ListTagsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.DefaultActivityOptions)

	var activityResult []*model.Tag
	a := tag.Tag{}
	err := workflow.ExecuteActivity(ctx, a.ListTags).Get(ctx, &activityResult)
	if err != nil {
		return nil, err
	}

	result := &tagtypes.ListTagsResult{
		Tags: activityResult,
	}

	return result, nil
}
