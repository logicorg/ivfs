package categorytypes

import (
	"cuelang.org/go/cuego"
	"github.com/logicorg/ivfs/internal/pkg/errors"
)

type RenameCategoryInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId"`
	Name            string `json:"name"`
}

type MoveCategoryInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId" cue:">0"`
	ParentID        *int64 `json:"parentId,omitempty"`
}

func (i *MoveCategoryInput) Validate() error {
	err := cuego.Validate(i)
	if err != nil {
		return errors.ValidationError{Err: err}
	}

	return nil
}
