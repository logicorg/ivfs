// Package value contains the activities to deal with asset values
package value

import (
	"context"
	"fmt"

	avt "github.com/logicorg/ivfs/finance/common/types/assetvaluetypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// DeleteAssetValue is the activity to delete an AssetValue
func (a *AssetValueActivity) DeleteAssetValue(ctx context.Context, params avt.DeleteAssetValueParams) (*avt.DeleteAssetValueResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID == 0 && params.AssetID == 0 {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "AssetID"}}
	}

	query := db.NewDelete().Model((*model.AssetValue)(nil))
	if params.ID > 0 {
		query.Where("id = ?", params.ID)
	} else {
		query = query.Where("asset_id = ?", params.AssetID)
	}

	_, err = query.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error deleting asset value(s): %w", err)
	}

	return &avt.DeleteAssetValueResult{}, nil
}
