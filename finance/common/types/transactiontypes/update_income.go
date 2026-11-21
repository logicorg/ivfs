package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// UpdateIncomeInput states what a journal entry is now: an income with
// these fields, whatever the entry was before. It has the fields of
// RecordIncomeInput but the idempotency key, since the entry keeps its own.
type UpdateIncomeInput struct {
	JournalEntryID         int64 `json:"journalEntryId" cue:">0" jsonschema:"the journal entry to correct"`
	BalanceLedgerAccountID int64 `json:"balanceLedgerAccountId" cue:">0" jsonschema:"the balance paid into, an asset or liability of a financial account"`
	CategoryID             int64 `json:"categoryId" cue:">0" jsonschema:"the income category it comes from"`
	// Amount is compared as a string, as a decimal is encoded as one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount received, positive"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it arrived, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note, such as the payer's name"`
	TagIDs      []int64         `json:"tagIds,omitempty" jsonschema:"the tags of the entry, each taken once"`
}

// Validate checks the input against its CUE constraints, its date, and its
// tag ids.
func (i *UpdateIncomeInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	if err := validateOccurredAt(i.OccurredAt); err != nil {
		return err
	}
	return jt.ValidateTagIDs(i.TagIDs)
}

// Income returns the income the entry becomes, as it would be recorded, without
// a key.
func (i *UpdateIncomeInput) Income() RecordIncomeInput {
	return RecordIncomeInput{
		BalanceLedgerAccountID: i.BalanceLedgerAccountID, CategoryID: i.CategoryID,
		Amount: i.Amount, OccurredAt: i.OccurredAt, Description: i.Description,
		TagIDs: i.TagIDs,
	}
}
