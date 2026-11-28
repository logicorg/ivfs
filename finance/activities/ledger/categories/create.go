package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// CreateCategory returns the category with the input's key, creating it when
// there is none. The key is the parent, the kind, and the name, compared
// case-insensitively and stored trimmed, in the casing it was first given. A
// category that already exists is returned unchanged, closed or not, so a
// retry or a second create of the same category is harmless.
func (a *Activities) CreateCategory(ctx context.Context, input ct.CreateCategoryInput) (*ct.CategoryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	input.Name = strings.TrimSpace(input.Name)
	if err := validateName(input.Name); err != nil {
		return nil, err
	}
	if !model.IsCategoryKind(input.Kind) {
		return nil, internalerrors.ErrInvalidParameter{Description: `category kind must be "income" or "expense"`}
	}
	if input.ParentID != nil && *input.ParentID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "parent category ID must be positive"}
	}

	existing, err := findCategory(ctx, db, input)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return &ct.CategoryResult{CategoryID: existing.ID}, nil
	}

	result := &ct.CategoryResult{}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		if input.ParentID != nil {
			locked, err := lockCategoriesTx(ctx, tx, *input.ParentID)
			if err != nil {
				return err
			}
			parent, ok := locked[*input.ParentID]
			if !ok {
				return internalerrors.ErrNotFound{ResourceType: resourceType, ID: *input.ParentID}
			}
			if err := checkParentTx(ctx, tx, parent, nil, input.Kind, 0); err != nil {
				return err
			}
		}

		category := &model.Category{ParentID: input.ParentID, Kind: input.Kind, Name: input.Name}
		if _, err := tx.NewInsert().Model(category).Returning("id").Exec(ctx); err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
		result.CategoryID = category.ID
		return nil
	})
	if conflict := sqllib.AlreadyExists(err, resourceType); conflict != nil {
		// Another create wrote the same key after the lookup above, and the
		// unique index held this insert until it committed. The violation
		// aborted the transaction, so the lookup runs after it.
		existing, err := findCategory(ctx, db, input)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, conflict
		}
		return &ct.CategoryResult{CategoryID: existing.ID}, nil
	}
	if err != nil {
		return nil, err
	}
	return result, nil
}

// findCategory returns the category with the input's key, or nil when there
// is none. A missing parent matches only a top-level category.
func findCategory(ctx context.Context, db bun.IDB, input ct.CreateCategoryInput) (*model.Category, error) {
	category := new(model.Category)
	query := db.NewSelect().Model(category).
		Where("cat.kind = ?", input.Kind).
		Where("lower(cat.name) = lower(?)", input.Name)
	if input.ParentID != nil {
		query = query.Where("cat.parent_id = ?", *input.ParentID)
	} else {
		query = query.Where("cat.parent_id IS NULL")
	}

	if err := query.Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("find category: %w", err)
	}
	return category, nil
}
