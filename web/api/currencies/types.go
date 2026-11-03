package currencies

import (
	"time"

	ct "github.com/logicorg/ivfs/finance/common/types/currencytypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Currency is a currency as the API returns it.
type Currency struct {
	ID        int64  `json:"id"`
	Code      string `json:"code" doc:"ISO 4217 code, such as EUR"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol,omitempty"`
	Precision int    `json:"precision" doc:"number of decimal places, such as 2 for EUR or 0 for JPY"`
}

// CurrencyList is every currency, in no particular order.
type CurrencyList struct {
	Currencies []Currency `json:"currencies"`
}

// ListCurrenciesOutput is the response of GET /api/currencies.
type ListCurrenciesOutput struct {
	Body CurrencyList
}

// CreateCurrencyInput is the request of POST /api/currencies.
type CreateCurrencyInput struct {
	Body ct.InsertCurrencyParams
}

// CreateCurrencyOutput is the response of POST /api/currencies: the id of
// the currency with that code, created now or before.
type CreateCurrencyOutput struct {
	Body ct.InsertCurrencyResult
}

// CurrencyChanges are the fields of a currency to change; an omitted field
// stays as it is.
type CurrencyChanges struct {
	Code   *string `json:"code,omitempty" doc:"ISO 4217 code, stored upper-cased"`
	Name   *string `json:"name,omitempty"`
	Symbol *string `json:"symbol,omitempty"`
}

// UpdateCurrencyInput is the request of PATCH /api/currencies/{id}.
type UpdateCurrencyInput struct {
	ID   int64 `path:"id" minimum:"1"`
	Body CurrencyChanges
}

// DeleteCurrencyInput is the request of DELETE /api/currencies/{id}.
type DeleteCurrencyInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// ListConversionRatesInput is the request of GET
// /api/currencies/{id}/rates/{quoteId}.
type ListConversionRatesInput struct {
	ID      int64 `path:"id" minimum:"1" doc:"the base currency"`
	QuoteID int64 `path:"quoteId" minimum:"1" doc:"the quote currency"`
	gateway.RangeInput
}

// ConversionRate is how many units of the quote currency one unit of the
// base was worth on a date.
type ConversionRate struct {
	Date       string         `json:"date" format:"date"`
	Rate       gateway.Amount `json:"rate"`
	ObservedAt time.Time      `json:"observedAt" doc:"when the rate was really observed: before date on a day that repeats an earlier one, such as a Saturday"`
}

// ProviderRates are one provider's rates, a date each, oldest first.
type ProviderRates struct {
	Provider string           `json:"provider"`
	Rates    []ConversionRate `json:"rates"`
}

// ConversionRates are every provider's rates of a pair, and the providers
// that did not answer.
type ConversionRates struct {
	Rates  []ProviderRates         `json:"rates"`
	Errors []gateway.ProviderError `json:"errors"`
}

// ListConversionRatesOutput is the response of GET
// /api/currencies/{id}/rates/{quoteId}.
type ListConversionRatesOutput struct {
	Body ConversionRates
}
