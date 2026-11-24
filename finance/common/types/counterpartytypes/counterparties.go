// Package counterpartytypes holds the inputs and results of the counterparty
// activities and workflows.
package counterpartytypes

import (
	"strings"

	"cuelang.org/go/cuego"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

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
