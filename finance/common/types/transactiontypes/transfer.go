package transactiontypes

import (
	"time"

	"cuelang.org/go/cuego"
	jt "github.com/logicorg/ivfs/finance/common/types/journaltypes"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// RecordTransferInput is money moved from one balance to another, such as
// from a current account to a savings vault, or from a EUR balance to a USD
// one (ADR 0008).
type RecordTransferInput struct {
	IdempotencyKey      string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per transfer meant, so a repeated request records it once"`
	FromLedgerAccountID int64  `json:"fromLedgerAccountId" cue:">0" jsonschema:"the balance the money leaves"`
	ToLedgerAccountID   int64  `json:"toLedgerAccountId" cue:">0" jsonschema:"the balance the money reaches"`
	// Amount and ToAmount are compared as strings, as a decimal is encoded as
	// one. A zero ToAmount is an omitted one.
	Amount      decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the amount that leaves, positive"`
	ToAmount    decimal.Decimal `json:"toAmount,omitzero" cue:"!~\"^-\"" jsonschema:"the amount that arrives, required when the balances are in different currencies; the amount that leaves if omitted"`
	OccurredAt  time.Time       `json:"occurredAt" jsonschema:"the instant it was moved, such as 2026-10-03T10:00:00Z"`
	Description string          `json:"description,omitempty" jsonschema:"a note"`
	TagIDs      []int64         `json:"tagIds,omitempty" jsonschema:"the tags of the entry, each taken once"`
	Charges     []Split         `json:"charges,omitempty" jsonschema:"what moving the money cost, such as an instalment's interest or a fee, taken out of the amount that leaves; at most one between currencies"`
}

// Validate checks the input against its CUE constraints, its date, its tag
// ids, its charges, and that the money moves between two different accounts.
func (i *RecordTransferInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	if err := validateOccurredAt(i.OccurredAt); err != nil {
		return err
	}
	if i.FromLedgerAccountID == i.ToLedgerAccountID {
		return errs.ErrInvalidParameter{Description: "a transfer moves money between two different balances"}
	}
	if err := validateCharges(i.Amount, i.Charges); err != nil {
		return err
	}
	return jt.ValidateTagIDs(i.TagIDs)
}
