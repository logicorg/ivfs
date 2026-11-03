package asset

import (
	"context"
	"fmt"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/internal/pkg/errors"
)

// UpdateAsset updates an asset in the DB
func (a *AssetActivity) UpdateAsset(
	ctx context.Context,
	params at.UpdateAssetParams,
) (*at.UpdateAssetResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	err = validateFindBy(params.FindBy)
	if err != nil {
		return nil, err
	}

	// retrieve
	retrieveResult, err := a.RetrieveAsset(ctx, params.FindBy)
	if err != nil {
		return nil, fmt.Errorf("error retrieving asset: %w", err)
	}

	if retrieveResult.Asset == nil {
		return nil, fmt.Errorf("unexpected nil asset")
	}

	// update
	if params.Name != nil {
		retrieveResult.Asset.Name = *params.Name
	}

	if params.UpdateDescription {
		retrieveResult.Asset.Description = params.Description
	}

	if params.UpdateTicker {
		retrieveResult.Asset.Ticker = params.Ticker
	}

	if params.CreatedAt != nil {
		retrieveResult.Asset.CreatedAt = *params.CreatedAt
	}

	// save
	_, err = db.NewUpdate().Model(retrieveResult.Asset).WherePK().Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("error updating asset: %w", err)
	}

	return &at.UpdateAssetResult{ID: retrieveResult.Asset.ID}, nil
}

func validateFindBy(fb at.RetrieveAssetParams) error {
	if fb.ID <= 0 && fb.Name == "" && fb.Ticker == "" {
		return errors.ErrInvalidParameter{Description: "at least one of ID, Name or Ticker must be provided to find the asset"}
	}
	return nil
}
