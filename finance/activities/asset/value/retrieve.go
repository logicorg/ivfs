package value

import (
	"context"
	"fmt"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// RetrieveAssetValue is the activity to retrieve a single AssetValue
func (a *AssetValueActivity) RetrieveAssetValue(ctx context.Context, params avt.RetrieveAssetValueParams) (*avt.RetrieveAssetValueResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID == 0 && params.AssetID == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "AssetID"}}
	}

	var assetValue model.AssetValue
	query := db.NewSelect().Model(&assetValue)

	if params.ID > 0 {
		query.Where("id = ?", params.ID)
	} else {
		query.Where("asset_id = ?", params.AssetID)
		if params.Date != nil {
			query.Where("observed_at <= ?", params.Date)
		}
		// The latest at or before the date, if any.
		query.Order("observed_at DESC").Limit(1)
	}

	err = query.Scan(ctx)
	if err != nil {
		if sqllib.IsNotFound(err) {
			return nil, errors.ErrNotFound{
				ResourceType: "AssetValue",
				Err:          err,
			}
		}
		return nil, fmt.Errorf("error retrieving asset value: %w", err)
	}

	return &avt.RetrieveAssetValueResult{AssetValue: &assetValue}, nil
}

// RetrieveAssetValues is the activity to retrieve multiple AssetValues
func (a *AssetValueActivity) RetrieveAssetValues(ctx context.Context, params avt.RetrieveAssetValuesParams) (*avt.RetrieveAssetValuesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.AssetID == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"AssetID"}}
	}

	assetValues := make([]model.AssetValue, 0)
	query := db.NewSelect().Model(&assetValues).Where("asset_id = ?", params.AssetID)

	if params.Since != nil {
		query.Where("observed_at >= ?", *params.Since)
	}

	err = query.Order("observed_at ASC").Scan(ctx)
	if err != nil {
		if sqllib.IsNotFound(err) {
			return &avt.RetrieveAssetValuesResult{AssetValues: []*model.AssetValue{}}, nil
		}
		return nil, fmt.Errorf("error retrieving asset values: %w", err)
	}

	resultAssetValues := make([]*model.AssetValue, len(assetValues))
	for i := range assetValues {
		resultAssetValues[i] = &assetValues[i]
	}

	return &avt.RetrieveAssetValuesResult{AssetValues: resultAssetValues}, nil
}
