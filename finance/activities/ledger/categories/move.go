package categories

import (
	"context"
	"fmt"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// MoveCategory puts a category under another parent, or at the top level. It
// is allowed only while the category and the ones below it have no postings.
// The new parent is open, of the same kind, without postings, and not the
// category or one below it, and the hierarchy stays within its depth. Moving
// a category to where it is succeeds, so a retry is harmless.
func (a *Activities) MoveCategory(ctx context.Context, input ct.MoveCategoryInput) (*ct.CategoryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		ids := []int64{input.CategoryID}
		if input.ParentID != nil {
			ids = append(ids, *input.ParentID)
		}
		locked, err := lockCategoriesTx(ctx, tx, ids...)
		if err != nil {
			return err
		}
		category, ok := locked[input.CategoryID]
		if !ok {
			return internalerrors.ErrNotFound{ResourceType: resourceType, ID: input.CategoryID}
		}
		if category.ParentID == nil && input.ParentID == nil {
			return nil
		}
		if category.ParentID != nil && input.ParentID != nil && *category.ParentID == *input.ParentID {
			return nil
		}

		subtree, height, err := subtreeTx(ctx, tx, category.ID)
		if err != nil {
			return err
		}
		if hasPostings, err := hasPostingsTx(ctx, tx, subtree); err != nil {
			return err
		} else if hasPostings {
			return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: "a category with entries, or with one below it that has, cannot be moved"}
		}

		if input.ParentID != nil {
			parent, ok := locked[*input.ParentID]
			if !ok {
				return internalerrors.ErrNotFound{ResourceType: resourceType, ID: *input.ParentID}
			}
			if err := checkParentTx(ctx, tx, parent, &category.ID, category.Kind, height); err != nil {
				return err
			}
		}

		category.ParentID, category.UpdatedAt = input.ParentID, time.Now().UTC()
		// The column is named, so a nil parent is written as NULL.
		if _, err := tx.NewUpdate().Model(category).Column("parent_id", "updated_at").WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, resourceType); conflict != nil {
				return conflict
			}
			return fmt.Errorf("move category: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ct.CategoryResult{CategoryID: input.CategoryID}, nil
}
