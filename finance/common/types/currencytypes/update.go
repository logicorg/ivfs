package currencytypes

// UpdateCurrencyParams holds the parameters for updating a currency.
type UpdateCurrencyParams struct {
	// ID is the ID of the currency in the database
	ID int64
	// Code is the new currency code. If nil it will stay unchanged
	Code *string
	// Name is the new currency name. If nil it will stay unchanged
	Name *string
	// Symbol is the new currency symbol. If nil it will stay unchanged
	Symbol *string
}
