package categorytypes

import (
	"fmt"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

// CheckKind fails unless the category is of the kind its posting is booked
// as, which would otherwise put an expense category on "Income" or the
// reverse.
func CheckKind(category *model.Category, kind model.LedgerAccountKind) error {
	if category.Kind == kind {
		return nil
	}
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q is an %s category, not an %s one", category.Name, category.Kind, kind),
	}
}
