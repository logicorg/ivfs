package categorytypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// CloseCategoryInput represents the input for closing a category (ledger account).
type CloseCategoryInput struct {
	LedgerAccountID int64     `json:"ledgerAccountId"`
	ClosedAt        time.Time `json:"closedAt"`
}

// DeleteCategoryInput deletes a category, moving its postings on kept entries
// to "Uncategorized", or merges it into ReplacementID, a category of the same
// kind and currency (ADR 0012). Deleting a category that does not exist is not
// an error.
type DeleteCategoryInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId" cue:">0"`
	ReplacementID   *int64 `json:"replacementId,omitempty" cue:">0"`
}

// Validate checks the command without the database.
func (i *DeleteCategoryInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// PreviewCategoryDeletionInput previews deleting a category (ADR 0012).
type PreviewCategoryDeletionInput struct {
	LedgerAccountID int64 `json:"ledgerAccountId" cue:">0"`
}

// Validate checks the input without the database.
func (i *PreviewCategoryDeletionInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}
