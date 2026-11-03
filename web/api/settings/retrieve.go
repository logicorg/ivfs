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

func registerRetrieve(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "retrieve-user-settings",
		Method:      http.MethodGet,
		Path:        "/api/settings",
		Summary:     "Retrieve the user's settings, the defaults if they have never changed them",
		Tags:        tags,
	}, func(ctx context.Context, _ *struct{}) (*RetrieveUserSettingsOutput, error) {
		result, err := gateway.RunWorkflow[st.RetrieveUserSettingsResult](ctx, c, settingswf.RetrieveUserSettings, st.RetrieveUserSettingsParams{})
		if err != nil {
			return nil, err
		}
		return &RetrieveUserSettingsOutput{Body: userSettings(result.Settings)}, nil
	})
}
