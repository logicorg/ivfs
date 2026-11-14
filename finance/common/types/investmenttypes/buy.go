package investmenttypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// BuyAssetInput is units of an asset bought with the cash of a brokerage
// account, such as 10 VWCE for 1,050 EUR with a 3 EUR fee (ADR 0009).
type BuyAssetInput struct {
	IdempotencyKey     string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per buy meant, so a repeated request records it once"`
	FinancialAccountID int64  `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account bought in"`
	AssetID            int64  `json:"assetId" cue:">0" jsonschema:"the asset bought"`
	// Units, Amount, and Fee are compared as strings, as a decimal is encoded
	// as one. A zero Fee is an omitted one.
	Units        decimal.Decimal `json:"units" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the units bought, positive"`
	Amount       decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the price paid for the units, before the fee, positive"`
	Fee          decimal.Decimal `json:"fee,omitzero" cue:"!~\"^-\"" jsonschema:"the broker's fee, zero or more, paid on top of the amount and part of the cost"`
	CurrencyCode string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the currency paid in, an ISO 4217 code such as EUR"`
	OccurredAt   time.Time       `json:"occurredAt" jsonschema:"the instant of the buy, such as 2026-10-03T10:00:00Z"`
	Description  string          `json:"description,omitempty" jsonschema:"a note"`
}

// Validate checks the input against its CUE constraints and its date.
func (i *BuyAssetInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	return i.rules()
}

// rules checks what CUE cannot, for a buy recorded or updated to.
func (i *BuyAssetInput) rules() error {
	return validateOccurredAt(i.OccurredAt)
}
