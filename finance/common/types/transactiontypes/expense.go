package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// RecordExpenseInput is money paid from a balance for an expense category,
// such as groceries paid by card (ADR 0008).
type RecordExpenseInput struct {
	IdempotencyKey          string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per expense meant, so a repeated request records it once"`
	BalanceLedgerAccountID  int64  `json:"balanceLedgerAccountId" cue:">0" jsonschema:"the balance paid from, an asset or liability of a financial account"`
	CategoryLedgerAccountID int64  `json:"categoryLedgerAccountId" cue:">0" jsonschema:"the expense category paid for"`
	// Amount is compared as a string, as a decimal is encoded as one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount paid, positive"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it was paid, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note, such as the shop's name"`
}

// Validate checks the input against its CUE constraints and its date.
func (i *RecordExpenseInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return validateOccurredAt(i.OccurredAt)
}
