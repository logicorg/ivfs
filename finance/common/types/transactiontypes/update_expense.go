package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// UpdateExpenseInput states what a journal entry is now: an expense with
// these fields, whatever the entry was before. It has the fields of
// RecordExpenseInput but the idempotency key, since the entry keeps its own.
type UpdateExpenseInput struct {
	JournalEntryID          int64 `json:"journalEntryId" cue:">0" jsonschema:"the journal entry to correct"`
	BalanceLedgerAccountID  int64 `json:"balanceLedgerAccountId" cue:">0" jsonschema:"the balance paid from, an asset or liability of a financial account"`
	CategoryLedgerAccountID int64 `json:"categoryLedgerAccountId" cue:">0" jsonschema:"the expense category paid for"`
	// Amount is compared as a string, as a decimal is encoded as one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount paid, positive"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it was paid, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note, such as the shop's name"`
}

// Validate checks the input against its CUE constraints and its date.
func (i *UpdateExpenseInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return validateOccurredAt(i.OccurredAt)
}

// Expense returns the expense the entry becomes, as it would be recorded, without
// a key.
func (i *UpdateExpenseInput) Expense() RecordExpenseInput {
	return RecordExpenseInput{
		BalanceLedgerAccountID: i.BalanceLedgerAccountID, CategoryLedgerAccountID: i.CategoryLedgerAccountID,
		Amount: i.Amount, OccurredAt: i.OccurredAt, Description: i.Description,
	}
}
