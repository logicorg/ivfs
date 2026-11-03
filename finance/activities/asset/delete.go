package asset

import (
	"context"
	"database/sql"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
)

// DeleteAsset is the activity to delete an Asset
func (a *AssetActivity) DeleteAsset(ctx context.Context, params at.DeleteAssetParams) (*at.DeleteAssetResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID == 0 && params.Name == "" && params.Ticker == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name", "Ticker"}}
	}

	query := db.NewDelete().Model((*model.Asset)(nil))
	switch {
	case params.ID > 0:
		query.Where("id = ?", params.ID)
	case params.Name != "":
		query = query.Where("name = ?", params.Name)
	case params.Ticker != "":
		query = query.Where("ticker = ?", params.Ticker)
	default:
		return nil, errors.ErrInvalidParameter{Description: "at least one of ID, Name or Ticker must be provided"}
	}

	_, err = query.Exec(ctx)
	if conflict := sqllib.StillReferenced(err, "Asset"); conflict != nil {
		return nil, conflict
	}
	switch err {
	case nil:
		return &at.DeleteAssetResult{}, nil
	case sql.ErrNoRows:
		return &at.DeleteAssetResult{}, nil
	default:
		return nil, err
	}
}
