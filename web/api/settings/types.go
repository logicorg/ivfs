package settings

import (
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// UserSettings are the user's settings, as the API returns and replaces them.
type UserSettings struct {
	DefaultCurrencyID *int64 `json:"defaultCurrencyId,omitempty" minimum:"1" doc:"the currency the net worth dashboard values in when the URL names none; omitted when the user has not chosen one"`
	// Only exclude is built; cost_basis joins the enum when it is (ADR 0015).
	GapPolicy string `json:"gapPolicy" enum:"exclude" doc:"how a holding with no value on a date is counted: exclude leaves it out of the total and flags the date"`
}

// userSettings returns s as the API lists it.
func userSettings(s *model.UserSettings) UserSettings {
	return UserSettings{DefaultCurrencyID: s.DefaultCurrencyID, GapPolicy: string(s.GapPolicy)}
}

// RetrieveUserSettingsOutput is the response of GET /api/settings.
type RetrieveUserSettingsOutput struct {
	Body UserSettings
}

// UpdateUserSettingsInput is the request of PUT /api/settings: every setting,
// so an omitted default currency clears it.
type UpdateUserSettingsInput struct {
	Body UserSettings
}

// params returns the workflow's params for in.
func (in *UpdateUserSettingsInput) params() st.UpdateUserSettingsParams {
	return st.UpdateUserSettingsParams{DefaultCurrencyID: in.Body.DefaultCurrencyID, GapPolicy: model.GapPolicy(in.Body.GapPolicy)}
}

// UpdateUserSettingsOutput is the response of PUT /api/settings: the
// settings as stored.
type UpdateUserSettingsOutput struct {
	Body UserSettings
}
