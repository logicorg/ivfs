package accounts

import (
	"fmt"

	it "github.com/logicorg/ivfs/finance/common/types/institutiontypes"
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"go.temporal.io/sdk/workflow"
)

// retrieveLedgerAccount returns the account a guarded change is about.
func retrieveLedgerAccount(ctx workflow.Context, id int64) (*model.LedgerAccount, error) {
	var result lt.RetrieveLedgerAccountResult
	if err := workflow.ExecuteActivity(ctx, la.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: id}).Get(ctx, &result); err != nil {
		return nil, err
	}
	return result.LedgerAccount, nil
}

// refuseReservedAccount fails when the account is a shared or stray account,
// which the ledger finds by its name and place (ADR 0012): renamed, moved, or
// closed, the next write that needs it would create another or fail.
func refuseReservedAccount(account *model.LedgerAccount, action string) error {
	if !lt.IsReservedLedgerAccount(account) {
		return nil
	}
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q (%s) is managed by the ledger and cannot be %s", account.Name, account.Kind, action),
	}
}

// refuseBecomingReserved fails when a change would make the account look like
// a shared or stray account, which the ledger would then take for its own.
func refuseBecomingReserved(account, changed *model.LedgerAccount, action string) error {
	if !lt.IsReservedLedgerAccount(changed) {
		return nil
	}
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q cannot be %s: a top-level %s account named %q is managed by the ledger; pick another name",
			account.Name, action, changed.Kind, changed.Name),
	}
}

// retrieveFinancialAccount returns the financial account a guarded change is
// about.
func retrieveFinancialAccount(ctx workflow.Context, id int64) (*model.FinancialAccount, error) {
	var result lt.RetrieveFinancialAccountResult
	if err := workflow.ExecuteActivity(ctx, la.RetrieveFinancialAccount, lt.RetrieveFinancialAccountParams{ID: id}).Get(ctx, &result); err != nil {
		return nil, err
	}
	return result.FinancialAccount, nil
}

// isUntrackedFinancialAccount reports whether an account of this name under
// the institution is the stray financial account (ADR 0012). The institution
// is retrieved only for the stray's name, which most accounts do not have.
func isUntrackedFinancialAccount(ctx workflow.Context, institutionID int64, name string) (bool, error) {
	if !lt.IsUntrackedFinancialAccountName(name) {
		return false, nil
	}
	var result it.RetrieveInstitutionResult
	if err := workflow.ExecuteActivity(ctx, ia.Retrieve, it.RetrieveInstitutionParams{ID: institutionID}).Get(ctx, &result); err != nil {
		return false, err
	}
	return lt.IsUntrackedInstitution(result.Institution.Name), nil
}

// untrackedFinancialAccountError is the error for a change to the stray
// financial account.
func untrackedFinancialAccountError(name, action string) error {
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q of the institution %q is managed by the ledger and cannot be %s",
			name, lt.UntrackedInstitutionName, action),
	}
}

// becomingUntrackedFinancialAccountError is the error for a change that would
// make a financial account the one the ledger takes for the stray.
func becomingUntrackedFinancialAccountError(name, action string) error {
	return errs.ErrInvalidParameter{
		Description: fmt.Sprintf("%q cannot be %s: the account %q of the institution %q is managed by the ledger; pick another name",
			name, action, lt.UntrackedAccountsName, lt.UntrackedInstitutionName),
	}
}
