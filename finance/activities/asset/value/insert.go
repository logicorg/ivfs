package value

import (
	"context"
	"fmt"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// InsertAssetValue is the activity to insert an AssetValue.
func (a *AssetValueActivity) InsertAssetValue(ctx context.Context, params avt.InsertAssetValueParams) (*avt.InsertAssetValueResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	params.Date = params.Date.UTC()

	assetValue := &model.AssetValue{}
	err = db.NewSelect().
		Model(assetValue).
		Where("observed_at = ?", params.Date).
		Where("value = ?", params.Value).
		Where("asset_id = ?", params.AssetID).
		Where("currency_id = ?", params.CurrencyID).
		Scan(ctx)

	switch {
	case err == nil:
		return &avt.InsertAssetValueResult{ID: assetValue.ID}, nil
	case sqllib.IsNotFound(err):
		// continue to insert
	default:
		return nil, fmt.Errorf("error checking for existing asset value: %w", err)
	}

	// if it does not exist, insert
	assetValue = &model.AssetValue{
		Value:      params.Value,
		CurrencyID: params.CurrencyID,
		AssetID:    params.AssetID,
		ObservedAt: params.Date,
	}

	_, err = db.NewInsert().Model(assetValue).Returning("id").Exec(ctx)
	if err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return nil, notFound
		}
		return nil, fmt.Errorf("error inserting asset value: %w", err)
	}

	return &avt.InsertAssetValueResult{ID: assetValue.ID}, nil
}
