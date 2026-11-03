package settingstypes

import "github.com/logicorg/ivfs/pkg/model"

// RetrieveUserSettingsParams selects nothing: a user has one set of settings.
// It exists because a transport always starts a workflow with one argument.
type RetrieveUserSettingsParams struct{}

// RetrieveUserSettingsResult holds the user's settings, the defaults when they
// have never changed them.
type RetrieveUserSettingsResult struct {
	Settings *model.UserSettings
}
