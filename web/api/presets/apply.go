package presets

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	pt "github.com/logicorg/ivfs/finance/common/types/presettypes"
	presetswf "github.com/logicorg/ivfs/finance/workflows/facade/presets"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerApply(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "apply-preset",
		Method:      http.MethodPost,
		Path:        "/api/presets/apply",
		Summary:     "Create the currencies, tags, and categories of a preset",
		Description: "Each item is created, or found if it exists, one at a time and each category before its children. " +
			"The whole preset is validated first, but the writes are not atomic: after a failure, applying the same preset again finds what was created and creates the rest.",
		Tags:   tags,
		Errors: []int{http.StatusUnprocessableEntity, http.StatusConflict},
	}, func(ctx context.Context, in *ApplyPresetInput) (*ApplyPresetOutput, error) {
		result, err := gateway.RunWorkflow[pt.ApplyPresetResult](ctx, c, presetswf.ApplyPreset, applyPresetInput(in.Body))
		if err != nil {
			return nil, err
		}
		return &ApplyPresetOutput{Body: AppliedPreset{
			CurrencyIDs: result.CurrencyIDs,
			TagIDs:      result.TagIDs,
			Categories:  newAppliedCategories(result.Categories),
		}}, nil
	})
}
