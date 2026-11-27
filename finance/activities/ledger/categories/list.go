package categories

import (
	"context"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// ListCategories returns the categories that match the filters, by name. The
// closed ones are left out unless asked for.
func (a *Activities) ListCategories(ctx context.Context, params ct.ListCategoriesParams) (*ct.ListCategoriesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.Kind != "" && !model.IsCategoryKind(params.Kind) {
		return nil, internalerrors.ErrInvalidParameter{Description: `category kind must be "income" or "expense"`}
	}

	categories := make([]model.Category, 0)
	query := db.NewSelect().Model(&categories).OrderExpr("lower(cat.name), cat.id")
	if params.ParentID != nil {
		query = query.Where("cat.parent_id = ?", *params.ParentID)
	}
	if params.Kind != "" {
		query = query.Where("cat.kind = ?", params.Kind)
	}
	if params.NameContains != "" {
		query = query.Where("cat.name ILIKE ?", "%"+params.NameContains+"%")
	}
	if !params.IncludeClosed {
		query = query.Where("cat.closed_at IS NULL")
	}

	if err := query.Scan(ctx); err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	return &ct.ListCategoriesResult{Categories: categories}, nil
}
