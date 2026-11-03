package assettypes

// DeleteAssetParams input type to delete asset operations
type DeleteAssetParams struct {
	ID     int64  `json:"id" jsonschema:"id of the asset to delete. Use either ID or Name or Ticker"`
	Name   string `json:"name" jsonschema:"name of the asset to delete. Use either ID or Name or Ticker"`
	Ticker string `json:"ticker" jsonschema:"ticker of the asset to delete. Use either ID or Name or Ticker"`
}

// DeleteAssetResult output type to delete asset operations
type DeleteAssetResult struct{}
