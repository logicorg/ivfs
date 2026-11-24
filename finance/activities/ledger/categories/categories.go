// Package categories holds the activities on ledger.category: the income and
// expense categories a posting may carry.
package categories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/internal/pkg/tenancy"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

type Activities struct{ Pools tenancy.Pools }

const (
	resourceType = "Category"
	// maxDepth is the number of parent links a category may have above it.
	maxDepth = 2
	// uncategorized is the label clients give the postings without a category.
	uncategorized = "uncategorized"
)

// validateName checks a trimmed name. "Uncategorized" is refused, compared as
// the unique key compares names, so the label of the postings without a
// category is never also a category.
func validateName(name string) error {
	if name == "" {
		return internalerrors.ErrMissingRequiredParameter{Parameters: []string{"Name"}}
	}
	if strings.ToLower(name) == uncategorized {
		return internalerrors.ErrInvalidParameter{Description: `"Uncategorized" is what the entries without a category are called, so a category needs another name`}
	}
	return nil
}

// loadCategory returns the category, or ErrNotFound.
func loadCategory(ctx context.Context, db bun.IDB, id int64) (*model.Category, error) {
	if id <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "category ID must be positive"}
	}

	category := new(model.Category)
	if err := db.NewSelect().Model(category).Where("cat.id = ?", id).Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, internalerrors.ErrNotFound{ResourceType: resourceType, ID: id}
		}
		return nil, fmt.Errorf("load category: %w", err)
	}
	return category, nil
}

// lockCategoriesTx locks the categories for the rest of the transaction and
// returns the ones that exist, by id. A posting shares the lock of its
// category before it looks for children, so what is decided here about the
// postings and the children of a locked category holds until the commit. The
// rows are locked in id order to avoid deadlocks.
func lockCategoriesTx(ctx context.Context, tx bun.Tx, ids ...int64) (map[int64]*model.Category, error) {
	var categories []*model.Category
	if err := tx.NewSelect().Model(&categories).Where("cat.id IN (?)", bun.List(ids)).OrderExpr("cat.id").For("NO KEY UPDATE OF cat").Scan(ctx); err != nil {
		return nil, fmt.Errorf("lock categories: %w", err)
	}

	byID := make(map[int64]*model.Category, len(categories))
	for _, category := range categories {
		byID[category.ID] = category
	}
	return byID, nil
}

// hasPostingsTx tells whether a posting carries any of the categories.
func hasPostingsTx(ctx context.Context, tx bun.Tx, ids []int64) (bool, error) {
	exists, err := tx.NewSelect().Model((*model.Posting)(nil)).Where("p.category_id IN (?)", bun.List(ids)).Exists(ctx)
	if err != nil {
		return false, fmt.Errorf("check category postings: %w", err)
	}
	return exists, nil
}

// childrenTx returns the categories directly under any of the given ones.
func childrenTx(ctx context.Context, tx bun.Tx, ids []int64) ([]model.Category, error) {
	var children []model.Category
	if err := tx.NewSelect().Model(&children).Where("cat.parent_id IN (?)", bun.List(ids)).OrderExpr("cat.id").Scan(ctx); err != nil {
		return nil, fmt.Errorf("load child categories: %w", err)
	}
	return children, nil
}

// subtreeTx returns the ids of a category and its descendants, and the number
// of levels below it.
func subtreeTx(ctx context.Context, tx bun.Tx, id int64) ([]int64, int, error) {
	ids, level, height := []int64{id}, []int64{id}, 0
	// A hierarchy is at most maxDepth deep; one more round only guards against
	// one that is not.
	for range maxDepth + 1 {
		children, err := childrenTx(ctx, tx, level)
		if err != nil {
			return nil, 0, err
		}
		if len(children) == 0 {
			break
		}
		height++
		level = level[:0]
		for _, child := range children {
			level = append(level, child.ID)
		}
		ids = append(ids, level...)
	}
	return ids, height, nil
}

// checkParentTx checks that a category of the given kind, with height levels
// below it, may go under parent, which the caller has locked. categoryID is
// the category when it already exists, and nil for a new one.
func checkParentTx(ctx context.Context, tx bun.Tx, parent *model.Category, categoryID *int64, kind model.LedgerAccountKind, height int) error {
	if parent.ClosedAt != nil {
		return internalerrors.ErrResourceClosed{ResourceType: resourceType, ID: parent.ID, Reason: fmt.Sprintf("parent category %d is closed", parent.ID)}
	}
	if parent.Kind != kind {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("parent category %d is of kind %s, and its children must be too", parent.ID, parent.Kind)}
	}
	if hasPostings, err := hasPostingsTx(ctx, tx, []int64{parent.ID}); err != nil {
		return err
	} else if hasPostings {
		return internalerrors.ErrResourceConflict{ResourceType: resourceType, Reason: fmt.Sprintf("category %d has entries, so it cannot have categories under it", parent.ID)}
	}

	// The walk up from the parent finds how deep it is, and the category
	// itself when the move would put it under one of its descendants.
	depth, current := 0, parent
	for {
		if categoryID != nil && current.ID == *categoryID {
			return internalerrors.ErrInvalidParameter{Description: "a category cannot go under itself or a category below it"}
		}
		if current.ParentID == nil || depth > maxDepth {
			break
		}
		depth++
		next, err := loadCategory(ctx, tx, *current.ParentID)
		if err != nil {
			return err
		}
		current = next
	}
	if depth+1+height > maxDepth {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("categories nest at most %d levels below a top-level one", maxDepth)}
	}
	return nil
}
