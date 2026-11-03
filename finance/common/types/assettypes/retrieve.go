package assettypes

import "github.com/logicorg/ivfs/pkg/model"

// RetrieveAssetParams structure holds fields by which you can retrieve an Asset.
type RetrieveAssetParams struct {
	ID     int64  `json:"id" jsonschema:"id of the asset to retrieve. Use either ID or Name or Ticker"`
	Name   string `json:"name" jsonschema:"name of the asset to retrieve. Use either ID or Name or Ticker"`
	Ticker string `json:"ticker" jsonschema:"ticker of the asset to retrieve. Use either ID or Name or Ticker"`
	ISIN   string `json:"isin" jsonschema:"isin of the asset to retrieve. Use either ID or Name or Ticker"`
}

// RetrieveAssetResult is the result of a RetrieveAsset operation.
type RetrieveAssetResult struct {
	Asset *model.Asset
}

// RetrieveAllAssetsResult is the result of a RetrieveAllAssets operation.
type RetrieveAllAssetsResult struct {
	Assets []*model.Asset
}
