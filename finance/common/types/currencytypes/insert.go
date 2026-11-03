package currencytypes

import (
	"errors"
	"strings"

	"cuelang.org/go/cuego"
	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// InsertCurrencyParams struct holds the params required to insert a currency in the DB.
type InsertCurrencyParams struct {
	// Code is the code of the currency (e.g., "USD", "EUR").
	Code string `json:"code" cue:"=~\"^[A-Z]{3}$\"" jsonschema:"the ISO 4217 code, such as EUR; stored upper-cased"`
	// Name is the name of the currency (e.g., "US Dollar", "Euro").
	Name string `json:"name" cue:"=~\"^[A-Za-z][A-Za-z ]*$\"" jsonschema:"the name of the currency, such as Euro"`
	// Symbol is the symbol of the currency (e.g., "$", "€").
	Symbol string `json:"symbol,omitempty" jsonschema:"the symbol of the currency, such as €; optional"`
	// Precision is the number of decimals the currency values have. Defaults to 2 if unspecified.
	Precision *int `json:"precision,omitempty" cue:">=0" jsonschema:"the number of decimal places, such as 2 for EUR or 0 for JPY; 2 if omitted"`
}

// Validate checks the parameters without the database. A code or name that is
// missing is reported as such, before the format checks that it would fail
// too.
func (i *InsertCurrencyParams) Validate() error {
	var missing []string
	if strings.TrimSpace(i.Code) == "" {
		missing = append(missing, "Code")
	}
	if strings.TrimSpace(i.Name) == "" {
		missing = append(missing, "Name")
	}
	if len(missing) > 0 {
		return errs.ErrMissingRequiredParameter{Parameters: missing}
	}

	err := cuego.Validate(i)
	if err != nil {
		return errs.ValidationError{Err: err}
	}

	if len([]rune(i.Symbol)) != len([]rune(strings.TrimSpace(i.Symbol))) {
		err := errors.New("invalid currency symbol: cannot have leading or trailing whitespace")
		return errs.ValidationError{Err: err}
	}

	if len([]rune(i.Symbol)) > 4 {
		err := errors.New("invalid currency symbol: must be 4 characters or less")
		return errs.ValidationError{Err: err}
	}

	return nil
}

// InsertCurrencyResult holds the result of the InsertCurrency operation.
// It contains the ID of the newly inserted currency.
type InsertCurrencyResult struct {
	CurrencyID int64 `json:"currencyId"`
}
