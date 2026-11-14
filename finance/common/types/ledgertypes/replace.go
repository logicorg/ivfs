package ledgertypes

import (
	"errors"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// ReplaceJournalEntryInput states what a posted entry is after a correction:
// its date, description, metadata, postings, and position movements. The entry
// keeps its id, its keys, its counterparty, and its times.
type ReplaceJournalEntryInput struct {
	JournalEntryID int64             `json:"journalEntryId"`
	OccurredAt     time.Time         `json:"occurredAt"`
	Description    *string           `json:"description,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Postings       []PostingInput    `json:"postings"`
	// Movements are the units the entry moves, and none for an entry that is
	// not a trade or a vest.
	Movements []ReplaceMovementInput `json:"movements,omitempty"`
}

// ReplaceMovementInput is one position movement of a replaced entry. It takes
// the entry's date and external id.
type ReplaceMovementInput struct {
	Kind               model.PositionMovementKind `json:"kind"`
	FinancialAccountID int64                      `json:"financialAccountId"`
	AssetID            int64                      `json:"assetId"`
	UnitsDelta         decimal.Decimal            `json:"unitsDelta"`
	// CostBasisAmount and CostBasisCurrencyID are given together or not at all,
	// as for a trade.
	CostBasisAmount     *decimal.Decimal `json:"costBasisAmount,omitempty"`
	CostBasisCurrencyID *int64           `json:"costBasisCurrencyId,omitempty"`
}

// ReplaceJournalEntryResult carries the ids of the movements written, in the
// order of the input, and the holdings that are negative after the replace,
// which was made regardless.
type ReplaceJournalEntryResult struct {
	JournalEntryID      int64             `json:"journalEntryId"`
	PositionMovementIDs []int64           `json:"positionMovementIds,omitempty"`
	Warnings            []NegativeHolding `json:"warnings,omitempty"`
}

// Entry returns the replaced fields as the input of a posting. The fields a
// replace keeps are left empty.
func (i *ReplaceJournalEntryInput) Entry() PostJournalEntryInput {
	return PostJournalEntryInput{OccurredAt: i.OccurredAt, Description: i.Description, Metadata: i.Metadata, Postings: i.Postings}
}

// Validate checks the command without the database: the entry as a posting
// does, and each movement as a trade or a vest does.
func (i *ReplaceJournalEntryInput) Validate() error {
	if i.JournalEntryID <= 0 {
		err := errors.New("ID must be greater than 0")
		return internalerrors.ValidationError{Err: err}
	}

	entry := i.Entry()
	if err := entry.Validate(); err != nil {
		return err
	}

	for _, m := range i.Movements {
		if !m.Kind.HasJournalEntry() {
			return internalerrors.ErrInvalidParameter{Description: "a movement of a journal entry is a trade or a vest"}
		}
		if m.FinancialAccountID <= 0 || m.AssetID <= 0 || m.UnitsDelta.IsZero() {
			return internalerrors.ErrInvalidParameter{Description: "a movement requires a financial account, an asset, and units other than zero"}
		}
		if (m.CostBasisAmount == nil) != (m.CostBasisCurrencyID == nil) {
			return internalerrors.ErrInvalidParameter{Description: "cost basis amount and currency must be given together"}
		}
		if m.CostBasisCurrencyID != nil && *m.CostBasisCurrencyID <= 0 {
			return internalerrors.ErrInvalidParameter{Description: "invalid cost basis currency"}
		}
	}
	return nil
}
