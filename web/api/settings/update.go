package settings

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	st "github.com/logicorg/ivfs/finance/common/types/settingstypes"
	settingswf "github.com/logicorg/ivfs/finance/workflows/settings"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerUpdate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "update-user-settings",
		Method:      http.MethodPut,
		Path:        "/api/settings",
		Summary:     "Replace the user's settings",
		Description: "Every setting is replaced: a body without defaultCurrencyId clears the default currency.",
		Tags:        tags,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *UpdateUserSettingsInput) (*UpdateUserSettingsOutput, error) {
		result, err := gateway.RunWorkflow[st.UpdateUserSettingsResult](ctx, c, settingswf.UpdateUserSettings, in.params())
		if err != nil {
			return nil, err
		}
		return &UpdateUserSettingsOutput{Body: userSettings(result.Settings)}, nil
	})
}
