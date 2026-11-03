// Package settings holds the workflows over a user's settings (ADR 0015).
package settings

import (
	"github.com/logicorg/ivfs/finance/activities/user/settings"
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/finance/workflows"
	"go.temporal.io/sdk/workflow"
)

// RetrieveUserSettings returns the user's settings, the defaults when they
// have never changed them.
func RetrieveUserSettings(ctx workflow.Context, _ st.RetrieveUserSettingsParams) (*st.RetrieveUserSettingsResult, error) {
	ctx = workflow.WithActivityOptions(ctx, workflows.GetDefaultActivityOptions())

	var sa *settings.Activities
	var result st.RetrieveUserSettingsResult
	if err := workflow.ExecuteActivity(ctx, sa.RetrieveUserSettings).Get(ctx, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
