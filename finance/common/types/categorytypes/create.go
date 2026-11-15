package categorytypes

import "github.com/logicorg/ivfs/pkg/model"

// CreateCategoryInput is the key of a category: its name, its kind, and its
// parent. A category has no currency.
type CreateCategoryInput struct {
	Name string `json:"name"`
	// ParentID is nil for a top-level category.
	ParentID *int64                  `json:"parentId,omitempty,omitzero" jsonschema:"the parent category, of the same kind, if any"`
	Kind     model.LedgerAccountKind `json:"kind" jsonschema:"income or expense"`
}

// CategoryResult names the category a command created or changed.
type CategoryResult struct {
	CategoryID int64 `json:"categoryId"`
}
