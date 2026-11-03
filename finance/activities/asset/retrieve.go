package asset

import (
	"context"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// RetrieveAsset retrieves an asset from the DB
func (a *AssetActivity) RetrieveAsset(
	ctx context.Context,
	params at.RetrieveAssetParams,
) (*at.RetrieveAssetResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID < 0 {
		return nil, errors.ErrInvalidParameter{Description: "ID must be greater than or equal to 0"}
	}

	var query *bun.SelectQuery
	switch {
	case params.ID > 0:
		query = model.FindAssetByIDQuery(db, params.ID)
	case params.Name != "":
		query = model.FindAssetByNameQuery(db, params.Name)
	case params.ISIN != "":
		query = model.FindAssetByISINQuery(db, params.ISIN)
	case params.Ticker != "":
		query = model.FindAssetByTickerQuery(db, params.Ticker)
	default:
		return nil, errors.ErrInvalidParameter{Description: "at least one asset identifier must be provided (ID, Name, ISIN, or Ticker)"}
	}

	retrieved := &model.Asset{}
	err = query.Scan(ctx, retrieved)
	if err != nil {
		if sqllib.IsNotFound(err) {
			return nil, errors.ErrNotFound{
				ResourceType: "Asset",
				Err:          err,
			}
		}
		return nil, err
	}

	return &at.RetrieveAssetResult{Asset: retrieved}, nil
}

// RetrieveAllAssets retrives all assets from the DB
func (a *AssetActivity) RetrieveAllAssets(ctx context.Context) ([]*model.Asset, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	assets := make([]*model.Asset, 0)

	err = db.NewSelect().Model((*model.Asset)(nil)).Scan(ctx, assets)
	if err != nil {
		return nil, err
	}
	return assets, nil
}
