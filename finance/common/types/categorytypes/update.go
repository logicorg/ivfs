package categorytypes

import (
	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
)

type RenameCategoryInput struct {
	CategoryID int64  `json:"categoryId" cue:">0"`
	Name       string `json:"name"`
}

// Validate checks the command without the database.
func (i *RenameCategoryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}

// MoveCategoryInput moves a category under ParentID, or to the top level when
// it is nil.
type MoveCategoryInput struct {
	CategoryID int64  `json:"categoryId" cue:">0"`
	ParentID   *int64 `json:"parentId,omitempty" cue:">0"`
}

// Validate checks the command without the database.
func (i *MoveCategoryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errors.ValidationError{Err: err}
	}
	return nil
}
