package assetvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

var (
	amountPattern   = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,8})?$`)
	currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
)

// wireValues is the answer as the spec puts it on the wire. The value stays
// a string until checked, so a JSON number, which may already have lost
// precision, is refused rather than parsed.
type wireValues struct {
	Values []struct {
		Date       string    `json:"date"`
		Value      string    `json:"value"`
		Currency   string    `json:"currency"`
		ObservedAt time.Time `json:"observedAt"`
	} `json:"values"`
}

// Values asks the provider for the value of asset on the dates of r.
func (c *Client) Values(ctx context.Context, asset model.Asset, r Range) ([]Value, error) {
	r, err := r.normalised()
	if err != nil {
		return nil, err
	}
	want := r.dates()
	notFound := errors.ErrNotFound{ResourceType: "AssetValue", ID: asset.Name}
	query := identifiers(asset)
	// The spec answers 400 to a request with neither, and no provider could
	// find the asset without one anyway.
	if !query.Has("isin") && !query.Has("ticker") {
		return nil, notFound
	}
	body, err := c.get(ctx, "values", query, r, notFound)
	if err != nil {
		return nil, err
	}

	var wire wireValues
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("the provider's answer is not values as the spec defines them: %w", err)
	}
	answered := make([]string, len(wire.Values))
	for i, w := range wire.Values {
		answered[i] = w.Date
	}
	dates, err := checkDates(answered, r, want)
	if err != nil {
		return nil, err
	}
	values := make([]Value, len(wire.Values))
	for i, w := range wire.Values {
		if !amountPattern.MatchString(w.Value) {
			return nil, fmt.Errorf("the provider answered the value %q for %s, which breaks the spec: "+
				"it must be a decimal string, not negative, with at most 8 decimal places", w.Value, w.Date)
		}
		if !currencyPattern.MatchString(w.Currency) {
			return nil, fmt.Errorf("the provider answered the currency %q for %s, which breaks the spec: "+
				"it must be an ISO 4217 code, such as USD", w.Currency, w.Date)
		}
		if err := checkObserved(dates[i], w.ObservedAt); err != nil {
			return nil, err
		}
		amount, err := decimal.NewFromString(w.Value)
		if err != nil {
			return nil, fmt.Errorf("parse the value %q: %w", w.Value, err)
		}
		values[i] = Value{Date: dates[i], Amount: amount, Currency: w.Currency, ObservedAt: w.ObservedAt}
	}
	return values, nil
}

// identifiers returns the query naming asset to a provider: what it has of
// the identifiers the spec knows, as the person entered them.
func identifiers(asset model.Asset) url.Values {
	query := url.Values{}
	for name, value := range map[string]*string{"isin": asset.ISIN, "ticker": asset.Ticker, "exchange": asset.Exchange} {
		if value != nil && strings.TrimSpace(*value) != "" {
			query.Set(name, *value)
		}
	}
	if asset.Kind != "" {
		query.Set("kind", string(asset.Kind))
	}
	return query
}
