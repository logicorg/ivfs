package asset

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// InsertAsset is an activity that inserts an asset in the database, or
// returns the one with its name unchanged. An asset with no kind is stored as
// other, since nothing reads an empty kind.
func (a *AssetActivity) InsertAsset(
	ctx context.Context,
	params at.InsertAssetParams,
) (*at.InsertAssetResult, error) {
	if strings.TrimSpace(params.Name) == "" {
		return nil, errors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}

	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	retrieved := &model.Asset{}
	err = model.FindAssetByNameQuery(db, params.Name).Scan(ctx, retrieved)
	switch err {
	case sql.ErrNoRows:
	case nil:
		return &at.InsertAssetResult{ID: retrieved.ID}, nil
	default:
		return nil, fmt.Errorf("failed to retrieve existing asset: %w", err)
	}

	kind := params.Kind
	if kind == "" {
		kind = model.AssetKindOther
	}
	asset := &model.Asset{
		Name:        params.Name,
		Kind:        kind,
		Description: params.Description,
		Ticker:      params.Ticker,
	}
	_, err = db.NewInsert().Model(asset).Returning("id").Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &at.InsertAssetResult{ID: asset.ID}, nil
}
