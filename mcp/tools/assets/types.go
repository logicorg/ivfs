package assets

// AssetSummary is a flat projection of model.Asset. The model itself cannot
// be used as an MCP tool's output type directly: Asset.Transactions and
// PositionMovement.Asset form a cycle, and jsonschema-go's reflection-based
// schema inference panics on a cyclic Go type rather than emitting a
// recursive schema.
type AssetSummary struct {
	ID       int64   `json:"id"`
	Name     string  `json:"name"`
	Kind     string  `json:"kind"`
	Ticker   *string `json:"ticker,omitempty"`
	ISIN     *string `json:"isin,omitempty"`
	Exchange *string `json:"exchange,omitempty"`
}

// ListAssetsToolResult is list_assets's output.
type ListAssetsToolResult struct {
	Assets []AssetSummary `json:"assets"`
}
