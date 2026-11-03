package settings

import (
	"github.com/logicorg/ivfs/finance/activities/user/settings"
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// UpdateUserSettings replaces the user's settings. Invalid settings are
// refused before any activity runs.
func UpdateUserSettings(ctx workflow.Context, params st.UpdateUserSettingsParams) (*st.UpdateUserSettingsResult, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var sa *settings.Activities
	var result st.UpdateUserSettingsResult
	if err := workflow.ExecuteActivity(ctx, sa.UpdateUserSettings, params).Get(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
