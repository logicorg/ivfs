// Package currencytypes holds common types to deal with currency
package currencytypes

// DeleteCurrencyParams holds the parameters for deleting a currency.
// You must provide either the ID or the Code of the currency to delete.
// If both are provided, the ID will be used.
type DeleteCurrencyParams struct {
	ID   int64
	Code string
}
