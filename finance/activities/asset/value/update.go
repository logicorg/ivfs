package value

import (
	"context"
	"fmt"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// UpdateAssetValue is the activity to update an AssetValue. It only allows updates by ID.
func (a *AssetValueActivity) UpdateAssetValue(ctx context.Context, params avt.UpdateAssetValueParams) (*avt.UpdateAssetValueResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID"}}
	}

	assetValue := &model.AssetValue{}
	err = db.NewSelect().Model(assetValue).Where("id = ?", params.ID).Scan(ctx)
	if err != nil {
		return nil, errors.ErrNotFound{ResourceType: "AssetValue", Err: err}
	}

	if params.Value != nil {
		assetValue.Value = *params.Value
	}
	if params.ObservedAt != nil {
		assetValue.ObservedAt = *params.ObservedAt
	}

	_, err = db.NewUpdate().Model(assetValue).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating asset value: %w", err)
	}

	return &avt.UpdateAssetValueResult{AssetValue: assetValue}, nil
}
