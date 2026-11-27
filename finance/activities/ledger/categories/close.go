package categories

import (
	"context"
	"fmt"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// CloseCategory closes a category, at ClosedAt or now if it is zero. Closing
// only stops a category from being offered: its postings stay, and it still
// takes new ones. It is idempotent: closing a closed category succeeds
// without changing ClosedAt. A category with open children is refused, so a
// closed parent never has an open category under it.
func (a *Activities) CloseCategory(ctx context.Context, input ct.CloseCategoryInput) error {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return err
	}

	if err := input.Validate(); err != nil {
		return err
	}
	closedAt := input.ClosedAt
	if closedAt.IsZero() {
		closedAt = time.Now().UTC()
	}

	return db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		// A new child locks its parent to check that it is open, so the lock
		// here makes the two wait for each other.
		locked, err := lockCategoriesTx(ctx, tx, input.CategoryID)
		if err != nil {
			return err
		}
		category, ok := locked[input.CategoryID]
		if !ok {
			return internalerrors.ErrNotFound{ResourceType: resourceType, ID: input.CategoryID}
		}
		if category.ClosedAt != nil {
			return nil
		}

		openChildren, err := tx.NewSelect().Model((*model.Category)(nil)).Where("cat.parent_id = ?", category.ID).Where("cat.closed_at IS NULL").Exists(ctx)
		if err != nil {
			return fmt.Errorf("check open child categories: %w", err)
		}
		if openChildren {
			return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: fmt.Sprintf("category %d has open categories under it; close them first", category.ID)}
		}

		category.ClosedAt, category.UpdatedAt = &closedAt, closedAt
		if _, err := tx.NewUpdate().Model(category).Column("closed_at", "updated_at").WherePK().Exec(ctx); err != nil {
			return fmt.Errorf("close category: %w", err)
		}
		return nil
	})
}
