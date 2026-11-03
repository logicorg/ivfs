package assets

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	assetswf "github.com/logicorg/ivfs/finance/workflows/assets"
	"github.com/logicorg/ivfs/web/api/gateway"
	"go.temporal.io/sdk/client"
)

func registerList(api huma.API, c client.Client) {
	huma.Register(api, huma.Operation{
		OperationID: "list-assets",
		Method:      http.MethodGet,
		Path:        "/api/assets",
		Summary:     "List the assets, by name",
		Description: "Both filters given must match. At most 500 assets are listed.",
		Tags:        tags,
	}, func(ctx context.Context, in *ListAssetsInput) (*ListAssetsOutput, error) {
		result, err := gateway.RunWorkflow[at.ListAssetsResult](ctx, c, assetswf.ListAssets, at.ListAssetsParams{
			NameContains: in.NameContains, TickerContains: in.TickerContains,
		})
		if err != nil {
			return nil, err
		}
		out := &ListAssetsOutput{Body: AssetList{Assets: make([]Asset, 0, len(result.Assets))}}
		for _, asset := range result.Assets {
			out.Body.Assets = append(out.Body.Assets, Asset{
				ID: asset.ID, Name: asset.Name, Kind: string(asset.Kind),
				Ticker: deref(asset.Ticker), ISIN: deref(asset.ISIN), Exchange: deref(asset.Exchange),
				Description: deref(asset.Description),
			})
		}
		return out, nil
	})
}

// deref is s, or empty if there is none, which the API omits.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
