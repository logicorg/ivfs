package ledgertypes

import (
	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// DeleteLedgerAccountInput deletes a ledger account, or merges it into
// ReplacementID, as ADR 0012 describes. Deleting an account that does not
// exist is not an error.
type DeleteLedgerAccountInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId" cue:">0"`
	ReplacementID   *int64 `json:"replacementId,omitempty" cue:">0"`
}

// Validate checks the command without the database.
func (i *DeleteLedgerAccountInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// DeleteFinancialAccountInput deletes a financial account with its ledger
// accounts, or merges it into ReplacementID, as ADR 0012 describes. Deleting
// an account that does not exist is not an error.
type DeleteFinancialAccountInput struct {
	FinancialAccountID int64  `json:"financialAccountId" cue:">0"`
	ReplacementID      *int64 `json:"replacementId,omitempty" cue:">0"`
}

// Validate checks the command without the database.
func (i *DeleteFinancialAccountInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// DeletionTarget is what one deletion empties: a ledger account, or a
// financial account with its ledger accounts. Exactly one of the two is set.
// The deletion activities share it, so a batch knows whose lines to move and
// where to.
type DeletionTarget struct {
	LedgerAccountID    *int64 `json:"ledgerAccountId,omitempty" cue:">0"`
	FinancialAccountID *int64 `json:"financialAccountId,omitempty" cue:">0"`
	ReplacementID      *int64 `json:"replacementId,omitempty" cue:">0"`
}

// Validate checks the target without the database.
func (t *DeletionTarget) Validate() error {
	if (t.LedgerAccountID == nil) == (t.FinancialAccountID == nil) {
		return errs.ErrInvalidParameter{Description: "a deletion target is either a ledger account or a financial account"}
	}
	if err := cuego.Validate(t); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// CloseAccountForDeletionResult reports whether the target no longer exists,
// so the deletion has nothing left to do.
type CloseAccountForDeletionResult struct {
	Gone bool `json:"gone"`
}

// DeletionBatchInput is one batch of a deletion: up to Limit entries, or
// movement groups, on Target.
type DeletionBatchInput struct {
	Target DeletionTarget `json:"target"`
	Limit  int            `json:"limit"`
}

// Validate checks the batch without the database.
func (i *DeletionBatchInput) Validate() error {
	if i.Limit <= 0 {
		return errs.ErrInvalidParameter{Description: "a deletion batch limit must be positive"}
	}
	return i.Target.Validate()
}

// DeletionBatchResult reports how many entries, or movement groups, a batch
// handled. None means the target has none left.
type DeletionBatchResult struct {
	Handled int `json:"handled"`
}

// DeleteAccountResult carries the holdings left negative by a merge, which
// was made regardless. A deletion changes no surviving holding, and the stray
// financial account's are never reported, so it carries none.
type DeleteAccountResult struct {
	Warnings []NegativeHolding `json:"warnings,omitempty"`
}
