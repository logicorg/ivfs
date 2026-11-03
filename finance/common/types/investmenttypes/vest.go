package investmenttypes

import (
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// VestInput is units of an asset received as compensation, such as 100 ACME
// RSUs vested at 10 USD each, of which the employer kept 40 for tax (ADR 0009).
type VestInput struct {
	IdempotencyKey     string `json:"idempotencyKey" cue:"!~\"^[[:space:]]*$\"" jsonschema:"one key per vest meant, so a repeated request records it once"`
	FinancialAccountID int64  `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account or stock plan the units are delivered to"`
	AssetID            int64  `json:"assetId" cue:">0" jsonschema:"the asset vested"`
	// UnitsVested, UnitsWithheld, and FairMarketValue are compared as strings,
	// as a decimal is encoded as one. A zero UnitsWithheld is an omitted one.
	UnitsVested     decimal.Decimal `json:"unitsVested" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the units vested, before any withholding, positive"`
	UnitsWithheld   decimal.Decimal `json:"unitsWithheld,omitzero" cue:"!~\"^-\"" jsonschema:"the units the employer kept for tax, zero or more and fewer than those vested"`
	FairMarketValue decimal.Decimal `json:"fairMarketValue" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the value of one unit on the vesting date, positive"`
	CurrencyCode    string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the currency of the fair market value, an ISO 4217 code such as USD"`
	// IncomeCategoryLedgerAccountID is credited with the value of every unit
	// vested, and TaxCategoryLedgerAccountID debited with that of the units
	// withheld. Without withheld units the tax category is not used, so a
	// form may send the one it remembers.
	IncomeCategoryLedgerAccountID int64     `json:"incomeCategoryLedgerAccountId" cue:">0" jsonschema:"the income category the compensation is booked to, such as RSU compensation"`
	TaxCategoryLedgerAccountID    int64     `json:"taxCategoryLedgerAccountId,omitzero" cue:">=0" jsonschema:"the expense category the withheld units are booked to, such as Tax withheld; required when units are withheld"`
	OccurredAt                    time.Time `json:"occurredAt" jsonschema:"the instant of the vest, such as 2026-10-03T10:00:00Z"`
	Description                   string    `json:"description,omitempty" jsonschema:"a note"`
}

// Validate checks the input against its CUE constraints and its date, then
// the withholding, which CUE cannot compare with the units vested.
func (i *VestInput) Validate() error {
	if err := cuego.Validate(i); err != nil {
		return errs.ValidationError{Err: err}
	}
	if err := validateOccurredAt(i.OccurredAt); err != nil {
		return err
	}
	if i.UnitsWithheld.GreaterThanOrEqual(i.UnitsVested) {
		return errs.ErrInvalidParameter{Description: "the units withheld for tax must be fewer than the units vested, so the vest delivers some"}
	}
	if i.UnitsWithheld.IsPositive() && i.TaxCategoryLedgerAccountID == 0 {
		return errs.ErrMissingRequiredParameter{Parameters: []string{"TaxCategoryLedgerAccountID"}}
	}
	return nil
}
