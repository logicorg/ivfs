package assets

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerCreate(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "create-asset",
		Method:      http.MethodPost,
		Path:        "/api/assets",
		Summary:     "Create an asset, or find the one with its name",
		Description: "An asset with the name already is returned as it is, its kind, ticker, and description unchanged.",
		Tags:        tags,
		Errors:      []int{http.StatusUnprocessableEntity},
	}, func(ctx context.Context, in *CreateAssetInput) (*CreateAssetOutput, error) {
		params := at.InsertAssetParams{Name: in.Body.Name, Kind: model.AssetKind(in.Body.Kind)}
		if in.Body.Ticker != "" {
			params.Ticker = &in.Body.Ticker
		}
		if in.Body.Description != "" {
			params.Description = &in.Body.Description
		}
		result, err := gateway.RunWorkflow[at.InsertAssetResult](ctx, c, assetswf.InsertAsset, params)
		if err != nil {
			return nil, err
		}
		return &CreateAssetOutput{Body: CreatedAsset{ID: result.ID}}, nil
	})
}
