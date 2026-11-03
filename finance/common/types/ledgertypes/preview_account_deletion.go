package ledgertypes

import (
	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// PreviewAccountDeletionInput names what a deletion would empty: a ledger
// account, a financial account with its ledger accounts, or an institution
// with its financial accounts. Exactly one is set.
type PreviewAccountDeletionInput struct {
	LedgerAccountID    *int64 `json:"ledgerAccountId,omitempty" cue:">0"`
	FinancialAccountID *int64 `json:"financialAccountId,omitempty" cue:">0"`
	InstitutionID      *int64 `json:"institutionId,omitempty" cue:">0"`
}

// Validate checks the input without the database.
func (i *PreviewAccountDeletionInput) Validate() error {
	set := 0
	for _, id := range []*int64{i.LedgerAccountID, i.FinancialAccountID, i.InstitutionID} {
		if id != nil {
			set++
		}
	}
	if set != 1 {
		return errs.ErrInvalidParameter{Description: "a deletion preview takes exactly one of a ledger account, a financial account, or an institution"}
	}
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// AccountDeletionPreview is what a deletion would do to the account as it is
// now (ADR 0012). Entries and movement groups recorded after it is read are
// handled without having been shown.
type AccountDeletionPreview struct {
	DeletedEntries        int `json:"deletedEntries"`
	KeptEntries           int `json:"keptEntries"`
	DeletedMovementGroups int `json:"deletedMovementGroups"`
	KeptMovementGroups    int `json:"keptMovementGroups"`
	// StrayAmounts is what the kept entries move onto each stray ledger account.
	StrayAmounts []StrayAmount `json:"strayAmounts,omitempty"`
	// StrayUnits is what the kept entries and groups move onto the stray
	// financial account, by asset.
	StrayUnits []StrayUnits `json:"strayUnits,omitempty"`
	// Conflicts are why the deletion would be refused. The rest of the preview
	// still describes what it would do once they are resolved.
	Conflicts []string `json:"conflicts,omitempty"`
}

// StrayAmount is what lands on one stray ledger account, debits minus
// credits.
type StrayAmount struct {
	Name         string                  `json:"name"`
	Kind         model.LedgerAccountKind `json:"kind"`
	CurrencyCode string                  `json:"currencyCode"`
	Amount       decimal.Decimal         `json:"amount"`
}

// StrayUnits is how many units of an asset land on the stray financial
// account.
type StrayUnits struct {
	AssetID   int64           `json:"assetId"`
	AssetName string          `json:"assetName"`
	Units     decimal.Decimal `json:"units"`
}
