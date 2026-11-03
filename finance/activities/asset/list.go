package asset

import (
	"context"
	"fmt"

	at "github.com/logicorg/ivfs/finance/common/types/assettypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// ListAssets searches assets by a case-insensitive substring of the asset's
// name and/or ticker, the way a caller resolving something like "apple" or
// "AAPL" from natural language would. A blank ListAssetsParams matches every
// asset, ordered by name.
func (a *AssetActivity) ListAssets(ctx context.Context, params at.ListAssetsParams) (*at.ListAssetsResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	// The model alias "as" is a reserved SQL keyword, so every raw reference
	// to it below must be double-quoted.
	assets := make([]*model.Asset, 0)
	query := db.NewSelect().Model(&assets).
		OrderExpr(`"as".name ASC`).
		Limit(at.MaxAssetsResult)
	if params.NameContains != "" {
		query = query.Where(`"as".name ILIKE ?`, "%"+params.NameContains+"%")
	}
	if params.TickerContains != "" {
		query = query.Where(`"as".ticker ILIKE ?`, "%"+params.TickerContains+"%")
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list assets: %w", err)
	}
	return &at.ListAssetsResult{Assets: assets}, nil
}
