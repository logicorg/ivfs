package investmenttypes

import (
	"fmt"
	"slices"
	"time"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// UpdateInvestmentInput states what a journal entry is now: an investment of
// a kind, with the facts of that kind in the field named after it, whatever
// the entry was before. The facts are those of RecordInvestmentInput but the
// idempotency key, since the entry keeps its own.
type UpdateInvestmentInput struct {
	JournalEntryID int64            `json:"journalEntryId" jsonschema:"the journal entry to correct"`
	Kind           Kind             `json:"kind" jsonschema:"the kind the entry becomes, which names the field holding its facts"`
	Buy            *UpdateBuyInput  `json:"buy,omitempty" jsonschema:"the facts of a buy, set when the kind is buy"`
	Sell           *UpdateSellInput `json:"sell,omitempty" jsonschema:"the facts of a sale, set when the kind is sell"`
	Vest           *UpdateVestInput `json:"vest,omitempty" jsonschema:"the facts of a vest, set when the kind is vest"`
}

// UpdateBuyInput is the facts of a buy, as BuyAssetInput has them.
type UpdateBuyInput struct {
	FinancialAccountID int64 `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account bought in"`
	AssetID            int64 `json:"assetId" cue:">0" jsonschema:"the asset bought"`
	// Units, Amount, and Fee are compared as strings, as a decimal is encoded
	// as one. A zero Fee is an omitted one.
	Units        decimal.Decimal `json:"units" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the units bought, positive"`
	Amount       decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the price paid for the units, before the fee, positive"`
	Fee          decimal.Decimal `json:"fee,omitzero" cue:"!~\"^-\"" jsonschema:"the broker's fee, zero or more, paid on top of the amount and part of the cost"`
	CurrencyCode string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the currency paid in, an ISO 4217 code such as EUR"`
	OccurredAt   time.Time       `json:"occurredAt" jsonschema:"the instant of the buy, such as 2026-10-03T10:00:00Z"`
	Description  string          `json:"description,omitempty" jsonschema:"a note"`
}

// UpdateSellInput is the facts of a sale, as SellAssetInput has them.
type UpdateSellInput struct {
	FinancialAccountID int64 `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account sold in"`
	AssetID            int64 `json:"assetId" cue:">0" jsonschema:"the asset sold"`
	// Units, Amount, and Fee are compared as strings, as a decimal is encoded
	// as one. A zero Fee is an omitted one.
	Units        decimal.Decimal `json:"units" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the units sold, positive"`
	Amount       decimal.Decimal `json:"amount" cue:"!=\"0\" & !~\"^-\"" jsonschema:"the price received for the units, before the fee, positive"`
	Fee          decimal.Decimal `json:"fee,omitzero" cue:"!~\"^-\"" jsonschema:"the broker's fee, zero or more and less than the amount, taken from the proceeds"`
	CurrencyCode string          `json:"currencyCode" cue:"!~\"^[[:space:]]*$\"" jsonschema:"the currency received, an ISO 4217 code such as EUR"`
	OccurredAt   time.Time       `json:"occurredAt" jsonschema:"the instant of the sale, such as 2026-10-03T10:00:00Z"`
	Description  string          `json:"description,omitempty" jsonschema:"a note"`
}

// UpdateVestInput is the facts of a vest, as VestInput has them.
type UpdateVestInput struct {
	FinancialAccountID int64 `json:"financialAccountId" cue:">0" jsonschema:"the brokerage account or stock plan the units are delivered to"`
	AssetID            int64 `json:"assetId" cue:">0" jsonschema:"the asset vested"`
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

// Validate checks the entry's id, that the kind is known, that its facts and
// no other kind's are set, and then the facts as recording them does.
func (i *UpdateInvestmentInput) Validate() error {
	if i.JournalEntryID <= 0 {
		return errs.ErrInvalidParameter{Description: "an update needs the id of the journal entry to correct"}
	}
	if !slices.Contains(kinds, i.Kind) {
		return errs.ErrInvalidParameter{Description: fmt.Sprintf("kind %q is not one of %v", i.Kind, kinds)}
	}

	set := 0
	for _, isSet := range []bool{i.Buy != nil, i.Sell != nil, i.Vest != nil} {
		if isSet {
			set++
		}
	}
	own := map[Kind]bool{KindBuy: i.Buy != nil, KindSell: i.Sell != nil, KindVest: i.Vest != nil}[i.Kind]
	if !own || set > 1 {
		return errs.ErrInvalidParameter{
			Description: fmt.Sprintf("an investment of kind %q needs its facts in the field %q, and no other kind's", i.Kind, i.Kind),
		}
	}

	// The constraints are those of the facts' own fields, and the rules CUE
	// cannot express are the ones recording the kind checks.
	switch i.Kind {
	case KindBuy:
		if err := cuego.Validate(i.Buy); err != nil {
			return errs.ValidationError{Err: err}
		}
		return i.Buy.Buy().rules()
	case KindSell:
		if err := cuego.Validate(i.Sell); err != nil {
			return errs.ValidationError{Err: err}
		}
		return i.Sell.Sell().rules()
	default:
		if err := cuego.Validate(i.Vest); err != nil {
			return errs.ValidationError{Err: err}
		}
		return i.Vest.Vest().rules()
	}
}

// Buy returns the buy the entry becomes, as it would be recorded, without a
// key.
func (i *UpdateBuyInput) Buy() *BuyAssetInput {
	return &BuyAssetInput{
		FinancialAccountID: i.FinancialAccountID,
		AssetID:            i.AssetID,
		Units:              i.Units,
		Amount:             i.Amount,
		Fee:                i.Fee,
		CurrencyCode:       i.CurrencyCode,
		OccurredAt:         i.OccurredAt,
		Description:        i.Description,
	}
}

// Sell returns the sale the entry becomes, as it would be recorded, without a
// key.
func (i *UpdateSellInput) Sell() *SellAssetInput {
	return &SellAssetInput{
		FinancialAccountID: i.FinancialAccountID,
		AssetID:            i.AssetID,
		Units:              i.Units,
		Amount:             i.Amount,
		Fee:                i.Fee,
		CurrencyCode:       i.CurrencyCode,
		OccurredAt:         i.OccurredAt,
		Description:        i.Description,
	}
}

// Vest returns the vest the entry becomes, as it would be recorded, without a
// key.
func (i *UpdateVestInput) Vest() *VestInput {
	return &VestInput{
		FinancialAccountID:            i.FinancialAccountID,
		AssetID:                       i.AssetID,
		UnitsVested:                   i.UnitsVested,
		UnitsWithheld:                 i.UnitsWithheld,
		FairMarketValue:               i.FairMarketValue,
		CurrencyCode:                  i.CurrencyCode,
		IncomeCategoryLedgerAccountID: i.IncomeCategoryLedgerAccountID,
		TaxCategoryLedgerAccountID:    i.TaxCategoryLedgerAccountID,
		OccurredAt:                    i.OccurredAt,
		Description:                   i.Description,
	}
}
