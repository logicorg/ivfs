package conversiontypes

import "time"

// InsertCurrencyValueParams input to an insert currency value operation
type InsertCurrencyValueParams struct {
	FromCurrencyID int64
	ToCurrencyID   int64
	ExchangeRate   float64
	EffectiveDate  time.Time
}

// InsertCurrencyValueResult result of an insert currency value operation
type InsertCurrencyValueResult struct {
	ID int64 // The ID of the newly inserted currency value
}
