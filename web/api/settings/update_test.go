package settings

import (
	"net/http"
	"testing"

	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	settingswf "github.com/logicorg/ivfs/finance/workflows/settings"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/apitest"
	"go.temporal.io/sdk/temporal"
)

// TestUpdate checks that the body replaces the settings, an omitted default
// currency clearing it, that the stored settings are returned, that a
// currency that does not exist is a 404, and that a body the types forbid
// runs no workflow.
func TestUpdate(t *testing.T) {
	apitest.RunCases(t, Register, map[string]apitest.Case{
		"set a default currency": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"defaultCurrencyId":2,"gapPolicy":"exclude"}`,
			Call: &apitest.WorkflowCall{
				Workflow: settingswf.UpdateUserSettings,
				Params:   st.UpdateUserSettingsParams{DefaultCurrencyID: new(int64(2)), GapPolicy: model.GapPolicyExclude},
				Result: st.UpdateUserSettingsResult{Settings: &model.UserSettings{
					ID: model.UserSettingsID, DefaultCurrencyID: new(int64(2)), GapPolicy: model.GapPolicyExclude,
				}},
			},
			WantStatus: http.StatusOK, WantBody: `{"defaultCurrencyId":2,"gapPolicy":"exclude"}`,
		},
		"clear the default currency": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"gapPolicy":"exclude"}`,
			Call: &apitest.WorkflowCall{
				Workflow: settingswf.UpdateUserSettings,
				Params:   st.UpdateUserSettingsParams{GapPolicy: model.GapPolicyExclude},
				Result:   st.UpdateUserSettingsResult{Settings: model.DefaultUserSettings()},
			},
			WantStatus: http.StatusOK, WantBody: `{"gapPolicy":"exclude"}`,
		},
		"a default currency that does not exist": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"defaultCurrencyId":9,"gapPolicy":"exclude"}`,
			Call: &apitest.WorkflowCall{
				Workflow: settingswf.UpdateUserSettings,
				Params:   st.UpdateUserSettingsParams{DefaultCurrencyID: new(int64(9)), GapPolicy: model.GapPolicyExclude},
				Err:      temporal.NewApplicationError("resource Currency not found with ID: 9", "ErrNotFound"),
			},
			WantStatus: http.StatusNotFound,
			WantBody:   apitest.Problem(http.StatusNotFound, "resource Currency not found with ID: 9"),
		},
		"a gap policy not built yet": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"gapPolicy":"cost_basis"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"no gap policy": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"defaultCurrencyId":2}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"a default currency id that is not positive": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"defaultCurrencyId":0,"gapPolicy":"exclude"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
		"an unknown field": {
			Method: http.MethodPut, Path: "/api/settings", Body: `{"gapPolicy":"exclude","currency":"EUR"}`,
			WantStatus: http.StatusUnprocessableEntity,
		},
	})
}
