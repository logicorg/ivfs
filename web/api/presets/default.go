package presets

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
)

func registerDefault(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-default-preset",
		Method:      http.MethodGet,
		Path:        "/api/presets/default",
		Summary:     "Get the default preset of currencies, tags, and categories",
		Description: "The same for every user. Nothing is created until a preset is applied.",
		Tags:        tags,
	}, func(context.Context, *DefaultPresetInput) (*DefaultPresetOutput, error) {
		return &DefaultPresetOutput{Body: newPreset(pt.Default())}, nil
	})
}
