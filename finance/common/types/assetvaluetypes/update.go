package assetvaluetypes

import (
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// UpdateAssetValueParams is the input to an update asset operation
type UpdateAssetValueParams struct {
	ID         int64
	ObservedAt *time.Time
	Value      *decimal.Decimal
}

// UpdateAssetValueResult is the result of an update asset value operation
type UpdateAssetValueResult struct {
	AssetValue *model.AssetValue
}
