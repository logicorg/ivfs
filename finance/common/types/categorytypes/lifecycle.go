package categorytypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// CloseCategoryInput closes a category, at ClosedAt or now if it is zero.
type CloseCategoryInput struct {
	CategoryID int64     `json:"categoryId" cue:">0"`
	ClosedAt   time.Time `json:"closedAt"`
}

// Validate checks the command without the database.
func (i *CloseCategoryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// DeleteCategoryInput deletes a category. Its postings lose their category,
// or take ReplacementID, which makes the deletion a merge. Deleting a category
// that does not exist is not an error.
type DeleteCategoryInput struct {
	CategoryID    int64  `json:"categoryId" cue:">0"`
	ReplacementID *int64 `json:"replacementId,omitempty" cue:">0"`
}

// Validate checks the command without the database.
func (i *DeleteCategoryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	if i.ReplacementID != nil && *i.ReplacementID == i.CategoryID {
		return errs.ErrInvalidParameter{Description: "a category cannot be merged into itself"}
	}
	return nil
}

// PreviewCategoryDeletionInput previews deleting a category.
type PreviewCategoryDeletionInput struct {
	CategoryID int64 `json:"categoryId" cue:">0"`
}

// Validate checks the input without the database.
func (i *PreviewCategoryDeletionInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// CategoryDeletionPreview is what deleting a category would change. No entry
// is deleted and no balance moves, so there is only one number to report.
type CategoryDeletionPreview struct {
	// Entries is the number of journal entries with a posting in the category,
	// which would lose it or take the replacement.
	Entries int `json:"entries"`
}
