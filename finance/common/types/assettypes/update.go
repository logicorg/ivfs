package assettypes

import "time"

// UpdateAssetParams is the input to an update asset operation
type UpdateAssetParams struct {
	FindBy RetrieveAssetParams

	Name              *string
	Description       *string
	UpdateDescription bool
	Ticker            *string
	UpdateTicker      bool

	CreatedAt *time.Time
}

// UpdateAssetResult is the result of an update asset operation
type UpdateAssetResult struct {
	ID int64
}
