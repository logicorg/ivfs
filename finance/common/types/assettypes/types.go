package assettypes

import (
	"github.com/logicorg/ivfs/pkg/model"
	"time"
)

// InsertAssetParams is the input to an insert Asset operation.
type InsertAssetParams struct {
	// Name is the name of the asset.
	Name string `json:"name" jsonschema:"name of the asset"`
	// Kind is what the asset is, such as etf; a blank one is stored as other.
	Kind model.AssetKind `json:"kind,omitempty" jsonschema:"what the asset is: equity, stock, etf, fund, bond, crypto, commodity, or other; other if omitted"`
	// Description is an optional description of the asset.
	Description *string `json:"description" jsonschema:"optional description of the asset"`
	// Ticker is an optional ticker symbol of the asset.
	Ticker *string `json:"ticker" jsonschema:"optional ticker symbol of the asset"`
}

type InsertAssetResult struct {
	ID int64 `json:"id" jsonschema:"id of the inserted asset"`
}

type GetAssetSharesAtDateParams struct {
	AssetID int64
	Date    time.Time
}

type GetAssetSharesAtDateResult struct {
	AssetSharesMap map[int64]int64
}

type GetAssetSharesForDateRangeParams struct {
	AssetID   int64
	StartDate time.Time
	EndDate   time.Time
}

type GetAssetSharesForDateRangeResult struct {
	Shares []float64
}

type GetAssetsSharesForDateRangeParams struct {
	AssetID   int64
	StartDate time.Time
	EndDate   time.Time
}

type GetAssetsSharesForDateRangeResult struct {
	Shares map[*model.Asset][]int64
}
