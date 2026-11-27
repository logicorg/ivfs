package categories

import (
	"context"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// PreviewCategoryDeletion reports how many journal entries DeleteCategory
// would change, and changes nothing. A category that does not exist is
// ErrNotFound.
func (a *Activities) PreviewCategoryDeletion(ctx context.Context, input ct.PreviewCategoryDeletionInput) (*ct.CategoryDeletionPreview, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}
	if _, err := loadCategory(ctx, db, input.CategoryID); err != nil {
		return nil, err
	}

	var entries int
	err = db.NewSelect().Model((*model.Posting)(nil)).ColumnExpr("COUNT(DISTINCT p.journal_entry_id)").Where("p.category_id = ?", input.CategoryID).Scan(ctx, &entries)
	if err != nil {
		return nil, fmt.Errorf("count entries of category: %w", err)
	}
	return &ct.CategoryDeletionPreview{Entries: entries}, nil
}
