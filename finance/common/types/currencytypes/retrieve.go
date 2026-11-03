package currencytypes

import (
	"errors"

	errs "github.com/logicorg/ivfs/internal/pkg/errors"
)

// RetrieveCurrencyParams holds the parameters for retrieving a currency.
type RetrieveCurrencyParams struct {
	ID     int64  `json:"id"`
	Name   string `json:"name"`
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

func (i *RetrieveCurrencyParams) Validate() error {
	if i.ID < 0 {
		err := errors.New("id cannot be negative")
		return errs.ValidationError{Err: err}
	}

	if i.ID == 0 && i.Name == "" && i.Code == "" && i.Symbol == "" {
		err := errors.New("at least one of id, name, code, or symbol must be supplied")
		return errs.ValidationError{Err: err}
	}

	return nil
}

// FindCurrencyByParams holds the parameters to find a currency.
type FindCurrencyByParams struct {
	Name   string `json:"name"`
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
}

func (i *FindCurrencyByParams) Validate() error {
	if i.Name == "" && i.Code == "" && i.Symbol == "" {
		err := errors.New("at least one of name, code, or symbol must be supplied")
		return errs.ValidationError{Err: err}
	}

	return nil
}
