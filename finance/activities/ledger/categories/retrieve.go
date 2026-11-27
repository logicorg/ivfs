package categories

import (
	"context"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
)

// RetrieveCategory returns a category, or ErrNotFound.
func (a *Activities) RetrieveCategory(ctx context.Context, params ct.RetrieveCategoryParams) (*ct.RetrieveCategoryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	category, err := loadCategory(ctx, db, params.ID)
	if err != nil {
		return nil, err
	}
	return &ct.RetrieveCategoryResult{Category: category}, nil
}
