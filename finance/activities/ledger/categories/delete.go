package categories

import (
	"context"
	"fmt"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// DeleteCategory deletes a category in one transaction. Its postings lose
// their category, or take the replacement, which makes the deletion a merge.
// No entry is deleted and no posting changes account, so no balance and no
// total per currency changes.
//
// The replacement is an open category of the same kind with no children; any
// other is ErrInvalidParameter. A category with children is
// ErrResourceConflict. Deleting a category that does not exist succeeds, so a
// retry is harmless.
func (a *Activities) DeleteCategory(ctx context.Context, input ct.DeleteCategoryInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		ids := []int64{input.CategoryID}
		if input.ReplacementID != nil {
			ids = append(ids, *input.ReplacementID)
		}
		// The lock makes a posting that is taking either category wait, so none
		// is left pointing at the deleted one or reaches a replacement that has
		// become a parent.
		locked, err := lockCategoriesTx(ctx, tx, ids...)
		if err != nil {
			return err
		}
		category, ok := locked[input.CategoryID]
		if !ok {
			return nil
		}

		children, err := childrenTx(ctx, tx, ids)
		if err != nil {
			return err
		}
		for _, child := range children {
			if *child.ParentID == category.ID {
				return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: fmt.Sprintf("category %d has categories under it; delete or move them first", category.ID)}
			}
		}

		if input.ReplacementID != nil {
			replacement, ok := locked[*input.ReplacementID]
			switch {
			case !ok:
				return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("replacement category %d does not exist", *input.ReplacementID)}
			case replacement.ClosedAt != nil:
				return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("replacement category %d is closed", replacement.ID)}
			case replacement.Kind != category.Kind:
				return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("replacement category %d is of kind %s, and category %d of kind %s", replacement.ID, replacement.Kind, category.ID, category.Kind)}
			case len(children) > 0:
				// The deleted category has none, so these are the replacement's.
				return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("replacement category %d has categories under it, and entries go in one of them", replacement.ID)}
			}
		}

		if _, err := tx.NewUpdate().Model((*model.Posting)(nil)).Set("category_id = ?", input.ReplacementID).Where("category_id = ?", category.ID).Exec(ctx); err != nil {
			return fmt.Errorf("move postings off category: %w", err)
		}
		if _, err := tx.NewDelete().Model(category).WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("delete category: %w", err)
		}
		return nil
	})
}
