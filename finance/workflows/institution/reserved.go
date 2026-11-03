package institution

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// retrieveInstitution returns the institution a guarded change is about.
func retrieveInstitution(ctx workflow.Context, findBy it.RetrieveInstitutionParams) (*model.Institution, error) {
	var result it.RetrieveInstitutionResult
	if err := workflow.ExecuteActivity(ctx, ia.Retrieve, findBy).Get(ctx, &result); err != nil {
		return nil, err
	}
	return result.Institution, nil
}

// refuseUntrackedInstitution fails when the institution is the one holding
// the stray financial account, which the ledger finds by its name (ADR 0012).
func refuseUntrackedInstitution(institution *model.Institution, action string) error {
	if !lt.IsUntrackedInstitution(institution.Name) {
		return nil
	}
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q is managed by the ledger and cannot be %s", institution.Name, action),
	}
}

// refuseUntrackedName fails when name would make an institution the one the
// ledger takes for the holder of the stray financial account.
func refuseUntrackedName(subject, name, action string) error {
	if !lt.IsUntrackedInstitution(name) {
		return nil
	}
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q cannot be %s: the institution %q is managed by the ledger; pick another name",
			subject, action, lt.UntrackedInstitutionName),
	}
}
