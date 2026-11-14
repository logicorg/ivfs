package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// UpdateIncomeInput states what a journal entry is now: an income with
// these fields, whatever the entry was before. It has the fields of
// RecordIncomeInput but the idempotency key, since the entry keeps its own.
type UpdateIncomeInput struct {
	JournalEntryID          int64 `json:"journalEntryId" cue:">0" jsonschema:"the journal entry to correct"`
	BalanceLedgerAccountID  int64 `json:"balanceLedgerAccountId" cue:">0" jsonschema:"the balance paid into, an asset or liability of a financial account"`
	CategoryLedgerAccountID int64 `json:"categoryLedgerAccountId" cue:">0" jsonschema:"the income category it comes from"`
	// Amount is compared as a string, as a decimal is encoded as one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount received, positive"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it arrived, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note, such as the payer's name"`
}

// Validate checks the input against its CUE constraints and its date.
func (i *UpdateIncomeInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return validateOccurredAt(i.OccurredAt)
}

// Income returns the income the entry becomes, as it would be recorded, without
// a key.
func (i *UpdateIncomeInput) Income() RecordIncomeInput {
	return RecordIncomeInput{
		BalanceLedgerAccountID: i.BalanceLedgerAccountID, CategoryLedgerAccountID: i.CategoryLedgerAccountID,
		Amount: i.Amount, OccurredAt: i.OccurredAt, Description: i.Description,
	}
}
