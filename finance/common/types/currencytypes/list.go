package currencytypes

import "github.com/logicorg/ivfs/pkg/model"

// ListCurrenciesParams selects the currencies to list. It has no filters:
// the table holds reference data small enough to return whole.
type ListCurrenciesParams struct{}

// ListCurrenciesResult holds every currency, in no particular order.
type ListCurrenciesResult struct {
	Currencies []*model.Currency `json:"currencies"`
}
