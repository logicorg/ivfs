package ledgertypes

import (
	"errors"
	"strings"
	"time"

	"cuelang.org/go/cuego"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

type CreateFinancialAccountInput struct {
	InstitutionID  int64   `json:"institutionId"`
	Name           string  `json:"name"`
	ExternalID     *string `json:"externalId,omitempty" jsonschema:"the provider's identifier of the account, such as an IBAN, if known"`
	CashCurrencyID *int64  `json:"cashCurrencyId,omitempty,omitzero" jsonschema:"if given, a Cash ledger account in this currency is created with the account"`
}

func (i *CreateFinancialAccountInput) Validate() error {
	if i.InstitutionID <= 0 || strings.TrimSpace(i.Name) == "" {
		err := errors.New("institutionId and name are required")
		return errs.ValidationError{Err: err}
	}

	if i.CashCurrencyID != nil && *i.CashCurrencyID <= 0 {
		err := errors.New("cashCurrencyId must be positive if provided")
		return errs.ValidationError{Err: err}
	}

	return nil
}

type CreateFinancialAccountResult struct {
	FinancialAccountID  int64  `json:"financialAccountId"`
	CashLedgerAccountID *int64 `json:"cashLedgerAccountId,omitempty"`
}

// RetrieveFinancialAccountParams holds the params to retrieve a FinancialAccount.
type RetrieveFinancialAccountParams struct {
	ID int64 `json:"id" cue:">0"`
}

func (p *RetrieveFinancialAccountParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type RetrieveFinancialAccountResult struct {
	FinancialAccount *model.FinancialAccount `json:"financialAccount"`
}

// MaxFinancialAccountsResult caps a ListFinancialAccounts search as a safety
// net rather than as pagination: a personal ledger has at most a few dozen
// financial accounts, so there is no cursor, only a fixed bound against a
// pathological result set.
const MaxFinancialAccountsResult = 500

// ListFinancialAccountsParams searches the FinancialAccount list, for
// example to resolve "revolut eur" to an id before a write command. A zero
// filter is not applied, and the filters that are given all have to match; a
// nil InstitutionID lists across every institution.
type ListFinancialAccountsParams struct {
	InstitutionID *int64 `json:"institutionId,omitempty" cue:">0"`
	// NameContains matches a case-insensitive substring of the account's own
	// name.
	NameContains string `json:"nameContains,omitempty"`
	// InstitutionNameContains matches a case-insensitive substring of the
	// owning institution's name, since accounts are commonly identified by
	// "institution + name" in speech (e.g. "revolut" for a "Revolut EUR"
	// account).
	InstitutionNameContains string `json:"institutionNameContains,omitempty"`
	IncludeClosed           bool   `json:"includeClosed,omitempty"`
}

func (p *ListFinancialAccountsParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

// ListFinancialAccountsResult is the (unpaginated) list of matching
// accounts, each with its institution and ledger accounts - with their
// currency - loaded, so a caller can disambiguate between candidates
// without a further round trip.
type ListFinancialAccountsResult struct {
	FinancialAccounts []model.FinancialAccount `json:"financialAccounts"`
}

// UpdateFinancialAccountMetadataInput updates the mutable metadata of a
// FinancialAccount. It never changes InstitutionID or the account's linked
// ledger accounts; a currency change requires a new financial/ledger account.
type UpdateFinancialAccountMetadataInput struct {
	FinancialAccountID int64   `json:"financialAccountId" cue:">0"`
	Name               *string `json:"name,omitempty"`
	ExternalID         *string `json:"externalId,omitempty"`
}

func (i *UpdateFinancialAccountMetadataInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type FinancialAccountMutationResult struct {
	FinancialAccountID int64 `json:"financialAccountId"`
}

// CloseFinancialAccountInput closes a FinancialAccount. Closing is idempotent
// and is refused while the account has active (non-closed) linked ledger
// accounts; those must be closed first.
type CloseFinancialAccountInput struct {
	FinancialAccountID int64     `json:"financialAccountId" cue:">0"`
	ClosedAt           time.Time `json:"closedAt"`
}

func (i *CloseFinancialAccountInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type CreateLedgerAccountInput struct {
	ParentID           *int64                  `json:"parentId,omitempty" jsonschema:"the parent ledger account, of the same kind and currency, if any"`
	FinancialAccountID *int64                  `json:"financialAccountId,omitempty" jsonschema:"the financial account the ledger account belongs to, if any"`
	CurrencyID         int64                   `json:"currencyId"`
	Name               string                  `json:"name"`
	Kind               model.LedgerAccountKind `json:"kind" jsonschema:"one of asset, liability, equity, income, expense"`
}

type CreateLedgerAccountResult struct {
	LedgerAccountID int64 `json:"ledgerAccountId"`
}

type RetrieveLedgerAccountParams struct {
	ID int64 `json:"id"`
}

type RetrieveLedgerAccountResult struct {
	LedgerAccount *model.LedgerAccount `json:"ledgerAccount"`
}

type ListLedgerAccountsParams struct {
	FinancialAccountID *int64 `json:"financialAccountId,omitempty" cue:">0"`
	ParentID           *int64 `json:"parentId,omitempty" cue:">0"`
	CurrencyID         *int64 `json:"currencyId,omitempty" cue:">0"`
	// NameContains matches a case-insensitive substring of the ledger
	// account's own name, for example to resolve "groceries" to a category
	// before a write command.
	NameContains string `json:"nameContains,omitempty"`
	// Kinds keeps only accounts of these kinds; empty matches every kind.
	Kinds         []model.LedgerAccountKind `json:"kinds,omitempty"`
	IncludeClosed bool                      `json:"includeClosed,omitempty"`
}

// Validate checks the ids against their CUE constraints, and that every
// requested kind is a known ledger account kind, which CUE cannot express.
func (p *ListLedgerAccountsParams) Validate() error {
	err := cuego.Validate(p)
	if err != nil {
		return errs.ValidationError{Err: err}
	}

	for _, kind := range p.Kinds {
		if !model.ValidLedgerAccountKind(kind) {
			err := errors.New("unknown ledger account kind " + string(kind))
			return errs.ValidationError{Err: err}
		}
	}

	return nil
}

type ListLedgerAccountsResult struct {
	LedgerAccounts []model.LedgerAccount `json:"ledgerAccounts"`
}

// RenameLedgerAccountInput represents the input for renaming a ledger account.
type RenameLedgerAccountInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId" cue:">0"`
	Name            string `json:"name" cue:"!~\"^[[:space:]]*$\""`
}

func (i *RenameLedgerAccountInput) Validate() error {
	err := cuego.Validate(i)
	if err != nil {
		return errs.ValidationError{Err: err}
	}

	return nil
}

type MoveLedgerAccountInput struct {
	LedgerAccountID int64  `json:"ledgerAccountId" cue:">0"`
	ParentID        *int64 `json:"parentId,omitempty"`
}

func (i *MoveLedgerAccountInput) Validate() error {
	err := cuego.Validate(i)

	if err != nil {
		return errs.ValidationError{Err: err}
	}

	return nil
}

type CloseLedgerAccountInput struct {
	LedgerAccountID int64     `json:"ledgerAccountId" cue:">0"`
	ClosedAt        time.Time `json:"closedAt"`
}

func (i *CloseLedgerAccountInput) Validate() error {
	err := cuego.Validate(i)
	if err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type LedgerAccountMutationResult struct {
	LedgerAccountID int64 `json:"ledgerAccountId"`
}

type CreateCounterpartyInput struct {
	Name       string  `json:"name"`
	ExternalID *string `json:"externalId,omitempty" jsonschema:"how an external source, such as a bank export, identifies the counterparty, if known"`
}

type CreateCounterpartyResult struct {
	CounterpartyID int64 `json:"counterpartyId"`
}

// RetrieveCounterpartyParams holds the params to retrieve a Counterparty.
type RetrieveCounterpartyParams struct {
	ID   int64  `json:"id" cue:">=0"`
	Name string `json:"name,omitempty"`
}

func (p *RetrieveCounterpartyParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return errs.ValidationError{Err: err}
	}

	if p.ID <= 0 && strings.TrimSpace(p.Name) == "" {
		err := errs.ErrMissingRequiredParameter{Parameters: []string{"ID", "Name"}}
		return errs.ValidationError{Err: err}
	}

	return nil
}

type RetrieveCounterpartyResult struct {
	Counterparty *model.Counterparty `json:"counterparty"`
}

// ListCounterpartiesParams searches the Counterparty list. A blank
// NameContains lists every counterparty.
type ListCounterpartiesParams struct {
	NameContains string `json:"nameContains,omitempty"`
}

type ListCounterpartiesResult struct {
	Counterparties []model.Counterparty `json:"counterparties"`
}

// RenameCounterpartyInput renames a Counterparty. It does not change any
// journal entry that references it.
type RenameCounterpartyInput struct {
	CounterpartyID int64  `json:"counterpartyId" cue:">0"`
	Name           string `json:"name" cue:"!~\"^[[:space:]]*$\""`
}

func (i *RenameCounterpartyInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type CounterpartyMutationResult struct {
	CounterpartyID int64 `json:"counterpartyId"`
}

// ClearCounterpartyReferencesInput names the counterparty that every journal
// entry, posted or not, stops referring to (ADR 0016).
type ClearCounterpartyReferencesInput struct {
	CounterpartyID int64 `json:"counterpartyId" cue:">0"`
}

func (i *ClearCounterpartyReferencesInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type ClearCounterpartyReferencesResult struct {
	// Cleared is the number of journal entries that lost the counterparty.
	Cleared int64 `json:"cleared"`
}

// DeleteCounterpartyInput deletes a Counterparty. The journal entries that
// refer to it keep their postings and lose their counterparty (ADR 0016).
type DeleteCounterpartyInput struct {
	CounterpartyID int64 `json:"counterpartyId" cue:">0"`
}

func (i *DeleteCounterpartyInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return nil
}

type DeleteCounterpartyResult struct {
	// Cleared is the number of journal entries that lost the counterparty.
	Cleared int64 `json:"cleared"`
}
