package model

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type Currency struct {
	bun.BaseModel `bun:"table:currency.currency,alias:c"`

	ID int64 `bun:"id,pk,autoincrement" json:"id" jsonschema:"id of the currency"`
	// Code is the ISO 4217 code of the currency.
	Code string `bun:"code,notnull,unique" json:"code" jsonschema:"ISO 4217 code of the currency"`
	// Name is the name of the currency.
	Name string `bun:"name,notnull,unique" json:"name" jsonschema:"name of the currency"`
	// Symbol is the symbol of the currency, if applicable.
	Symbol string `bun:"symbol" json:"symbol,omitempty" jsonschema:"symbol of the currency, if applicable"`
	// Precision is the number of decimal places the currency uses. For example, the Euro has a precision of 2, meaning that amounts has a cent as the smallest unit.
	Precision int `bun:"precision,notnull" json:"precision" jsonschema:"number of decimal places the currency uses."`

	// CreatedAt is the timestamp when the currency was created.
	CreatedAt time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp" json:"createdAt" jsonschema:"timestamp when the currency was created"`
	// UpdatedAt is the timestamp when the currency was last updated.
	UpdatedAt time.Time `bun:"updated_at,nullzero,notnull,default:current_timestamp" json:"updatedAt" jsonschema:"timestamp when the currency was last updated"`
}

func (c Currency) String() string {
	return c.Code
}

func FindCurrencyByCodeQuery(db bun.IDB, code string) *bun.SelectQuery {
	return db.NewSelect().Model((*Currency)(nil)).Where("c.code = ?", code)
}

func FindCurrencyByIDQuery(db bun.IDB, id int64) *bun.SelectQuery {
	return db.NewSelect().Model((*Currency)(nil)).Where("c.id = ?", id)
}

// Conversion is an observed exchange rate at a particular time. Rates are
// stored independently from currency display precision.
type Conversion struct {
	bun.BaseModel `bun:"table:currency.conversion,alias:conv"`

	ID             int64           `bun:"id,pk,autoincrement" json:"id"`
	FromCurrencyID int64           `bun:"from_currency_id,notnull" json:"fromCurrencyId"`
	ToCurrencyID   int64           `bun:"to_currency_id,notnull" json:"toCurrencyId"`
	Rate           decimal.Decimal `bun:"rate,type:numeric(36,18),notnull" json:"rate"`
	ObservedAt     time.Time       `bun:"observed_at,nullzero,notnull,default:current_timestamp" json:"observedAt"`
	Source         *string         `bun:"source" json:"source,omitempty" jsonschema:"source of the exchange rate"`

	FromCurrency *Currency `bun:"rel:belongs-to,join:from_currency_id=id,on_delete:restrict" json:"fromCurrency,omitempty"`
	ToCurrency   *Currency `bun:"rel:belongs-to,join:to_currency_id=id,on_delete:restrict" json:"toCurrency,omitempty"`
}

func FindConversionByUniquePropertiesQuery(db *bun.DB, fromCurrencyID, toCurrencyID int64, observedAt time.Time) *bun.SelectQuery {
	return db.NewSelect().Model((*Conversion)(nil)).
		Where("conv.from_currency_id = ?", fromCurrencyID).
		Where("conv.to_currency_id = ?", toCurrencyID).
		Where("conv.observed_at = ?", observedAt)
}

func DeleteConversionByUniquePropertiesQuery(db *bun.DB, fromCurrencyID, toCurrencyID int64, observedAt time.Time) *bun.DeleteQuery {
	return db.NewDelete().Model((*Conversion)(nil)).
		Where("from_currency_id = ?", fromCurrencyID).
		Where("to_currency_id = ?", toCurrencyID).
		Where("observed_at = ?", observedAt)
}
