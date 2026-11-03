package investmenttypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// SellAssetInput is units of an asset sold for cash in a brokerage account,
// such as 4 VWCE for 480 EUR with a 3 EUR fee (ADR 0009).
type SellAssetInput struct {
	IdempotencyKey     string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per sale meant, so a repeated request records it once"`
	FinancialAccountID int64  `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account sold in"`
	AssetID            int64  `json:"assetId" cue:">0" jsonschema:"the asset sold"`
	// Units, Amount, and Fee are compared as strings, as a decimal is encoded
	// as one. A zero Fee is an omitted one.
	Units        decimal.Decimal `json:"units" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the units sold, positive"`
	Amount       decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the price received for the units, before the fee, positive"`
	Fee          decimal.Decimal `json:"fee,omitzero" cue:"!~\"^-\"" jsonschema:"the broker's fee, zero or more and less than the amount, taken from the proceeds"`
	CurrencyCode string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the currency received, an ISO 4217 code such as EUR"`
	OccurredAt   time.Time       `json:"occurredAt" jsonschema:"the instant of the sale, such as 2026-10-03T10:00:00Z"`
	Description  string          `json:"description,omitempty" jsonschema:"a note"`
}

// Validate checks the input against its CUE constraints, its date, and that
// the fee leaves some proceeds, which CUE cannot compare as decimals.
func (i *SellAssetInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	if err := validateOccurredAt(i.OccurredAt); err != nil {
		return err
	}
	if i.Fee.GreaterThanOrEqual(i.Amount) {
		return errs.ErrInvalidParameter{Description: "the fee of a sale must be less than its amount, so the sale leaves some proceeds"}
	}
	return nil
}
