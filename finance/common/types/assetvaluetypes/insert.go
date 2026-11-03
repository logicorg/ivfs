package assetvaluetypes

import (
	"time"

	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// InsertAssetValueParams is the input for an insert asset value operation
type InsertAssetValueParams struct {
	AssetID    int64     `cue:">0"`
	CurrencyID int64     `cue:">0"`
	Date       time.Time `cue:"!=\"\""`
	Value      decimal.Decimal
}

func (p *InsertAssetValueParams) Validate() error {
	err := cuego.Validate(p)
	if err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}

// InsertAssetValueResult is the result of an insert asset value operation
type InsertAssetValueResult struct {
	ID int64
}
