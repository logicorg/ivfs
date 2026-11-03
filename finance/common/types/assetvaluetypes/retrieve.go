// Package assetvaluetypes contains common types Workflows and Activities use to deal with asset values
package assetvaluetypes

import (
	"time"

	"github.com/logicorg/ivfs/pkg/model"
)

// RetrieveAssetValueParams defines the parameters for retrieving a single asset value.
type RetrieveAssetValueParams struct {
	ID      int64
	AssetID int64
	Date    *time.Time
}

// RetrieveAssetValueResult defines the result of retrieving a single asset value.
type RetrieveAssetValueResult struct {
	AssetValue *model.AssetValue
}

// RetrieveAssetValuesParams defines the parameters for retrieving multiple asset values.
type RetrieveAssetValuesParams struct {
	AssetID int64
	Since   *time.Time
}

// RetrieveAssetValuesResult defines the result of retrieving multiple asset values.
type RetrieveAssetValuesResult struct {
	AssetValues []*model.AssetValue
}
