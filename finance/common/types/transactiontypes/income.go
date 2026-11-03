package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// RecordIncomeInput is money from an income category arriving in a balance,
// such as a salary paid into a bank account (ADR 0008).
type RecordIncomeInput struct {
	IdempotencyKey          string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per income meant, so a repeated request records it once"`
	BalanceLedgerAccountID  int64  `json:"balanceLedgerAccountId" cue:">0" jsonschema:"the balance paid into, an asset or liability of a financial account"`
	CategoryLedgerAccountID int64  `json:"categoryLedgerAccountId" cue:">0" jsonschema:"the income category it comes from"`
	// Amount is compared as a string, as a decimal is encoded as one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount received, positive"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it arrived, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note, such as the payer's name"`
}

// Validate checks the input against its CUE constraints and its date.
func (i *RecordIncomeInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return validateOccurredAt(i.OccurredAt)
}
