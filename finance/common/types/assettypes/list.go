package assettypes

import "github.com/logicorg/ivfs/pkg/model"

// MaxAssetsResult caps a ListAssets search as a safety net rather than as
// pagination, mirroring ListFinancialAccountsParams: a personal ledger holds
// at most a few hundred assets, so there is no cursor, only a fixed bound
// against a pathological result set.
const MaxAssetsResult = 500

// ListAssetsParams searches the Asset list, for example to resolve "apple"
// or "AAPL" to an id before a write command, the way RetrieveAssetParams's
// exact Name/Ticker/ISIN match cannot. A blank filter matches every asset;
// the filters given are ANDed together.
type ListAssetsParams struct {
	// NameContains matches a case-insensitive substring of the asset's name.
	NameContains string `json:"nameContains,omitempty" jsonschema:"case-insensitive substring match on the asset's name"`
	// TickerContains matches a case-insensitive substring of the asset's
	// ticker.
	TickerContains string `json:"tickerContains,omitempty" jsonschema:"case-insensitive substring match on the asset's ticker"`
}

// ListAssetsResult is the (unpaginated) list of matching assets.
type ListAssetsResult struct {
	Assets []*model.Asset `json:"assets"`
}
