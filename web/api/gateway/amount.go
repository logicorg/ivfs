package gateway

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/shopspring/decimal"
)

// Amount is a decimal amount in the API. It is a string on the wire, as
// decimal.Decimal marshals it, so it stays exact: the UI never turns it into
// a JavaScript number (ADR 0007).
type Amount struct {
	decimal.Decimal
}

// NewAmount returns d as an Amount.
func NewAmount(d decimal.Decimal) Amount {
	return Amount{Decimal: d}
}

// Schema describes an Amount as a decimal string, since decimal.Decimal has
// no schema of its own.
func (Amount) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Pattern:     `^-?[0-9]+(\.[0-9]+)?$`,
		Description: "a decimal amount, as a string so it stays exact",
		Examples:    []any{"1234.56"},
	}
}
