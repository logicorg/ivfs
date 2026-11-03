// Package assetvalue is the client side of the asset value providers (ADR
// 0014): the value of one unit of an asset, or the rate between two
// currencies, on the dates of a range, according to a source, such as the
// values entered by hand or a service implementing
// docs/specs/asset-value-provider.v1.yaml.
package assetvalue

import (
	"context"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// Value is what one unit of an asset was worth on a date, according to a
// provider.
type Value struct {
	// Date is one of the range's dates, as midnight UTC.
	Date   time.Time
	Amount decimal.Decimal
	// Currency is an ISO 4217 code, since an external provider knows nothing
	// of a user's currency IDs.
	Currency string
	// ObservedAt is when the value was really observed, which is before Date
	// on a date that repeats an earlier value, such as a Saturday.
	ObservedAt time.Time
}

// Rate is how many units of the quote currency one unit of the base
// currency was worth on a date, according to a provider.
type Rate struct {
	// Date is one of the range's dates, as midnight UTC.
	Date       time.Time
	Rate       decimal.Decimal
	ObservedAt time.Time
}

// Provider is a source of asset values and exchange rates. Both methods
// answer the dates of r from the first one the provider has a value for,
// oldest first; the value on a date is the latest one observed before the
// UTC day ends. A provider with no value on any date, including one that
// has no values or rates at all, returns errors.ErrNotFound, which is an
// ordinary answer rather than a failure.
type Provider interface {
	Values(ctx context.Context, asset model.Asset, r Range) ([]Value, error)
	// Rates takes base and quote as ISO 4217 codes.
	Rates(ctx context.Context, base, quote string, r Range) ([]Rate, error)
}

// Day returns the date of t, in t's own location, as midnight UTC, the form
// Value.Date takes.
func Day(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
