package categories

import (
	"context"
	"fmt"
	"strings"
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/categorytypes"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/uptrace/bun"
)

// RenameCategory gives a category another name, stored trimmed. A sibling of
// the same kind that holds the name, in any casing, is ErrResourceConflict.
// Renaming to the name it has succeeds, so a retry is harmless.
func (a *Activities) RenameCategory(ctx context.Context, input ct.RenameCategoryInput) (*ct.CategoryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}
	input.Name = strings.TrimSpace(input.Name)
	if err := validateName(input.Name); err != nil {
		return nil, err
	}

	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		category, err := loadCategory(ctx, tx, input.CategoryID)
		if err != nil {
			return err
		}
		category.Name, category.UpdatedAt = input.Name, time.Now().UTC()
		if _, err := tx.NewUpdate().Model(category).Column("name", "updated_at").WherePK().Exec(ctx); err != nil {
			if conflict := sqllib.AlreadyExists(err, resourceType); conflict != nil {
				return conflict
			}
			return fmt.Errorf("rename category: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &ct.CategoryResult{CategoryID: input.CategoryID}, nil
}
