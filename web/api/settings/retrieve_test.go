package settings

import (
	"net/http"
	"testing"

	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	settingswf "github.com/logicorg/ivfs/finance/workflows/settings"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
)

// TestRetrieve checks that the settings are returned as the API's own type,
// and that no default currency is omitted rather than null.
func TestRetrieve(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"settings with a default currency": {
			Method: http.MethodGet, Path: "/api/settings",
			Call: &apitest.WorkflowCall{
				Workflow: settingswf.RetrieveUserSettings, Params: st.RetrieveUserSettingsParams{},
				Result: st.RetrieveUserSettingsResult{Settings: &model.UserSettings{
					ID: model.UserSettingsID, DefaultCurrencyID: new(int64(2)), GapPolicy: model.GapPolicyExclude,
				}},
			},
			WantStatus: http.StatusOK, WantBody: `{"defaultCurrencyId":2,"gapPolicy":"exclude"}`,
		},
		"the defaults": {
			Method: http.MethodGet, Path: "/api/settings",
			Call: &apitest.WorkflowCall{
				Workflow: settingswf.RetrieveUserSettings, Params: st.RetrieveUserSettingsParams{},
				Result: st.RetrieveUserSettingsResult{Settings: model.DefaultUserSettings()},
			},
			WantStatus: http.StatusOK, WantBody: `{"gapPolicy":"exclude"}`,
		},
	})
}
