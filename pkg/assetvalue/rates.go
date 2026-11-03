package assetvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// ratePattern allows the 18 decimal places of model.ExchangeRateScale.
var ratePattern = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,18})?$`)

// wireRates is the answer as the spec puts it on the wire, the rate a
// string for the reason wireValues gives.
type wireRates struct {
	Rates []struct {
		Date       string    `json:"date"`
		Rate       string    `json:"rate"`
		ObservedAt time.Time `json:"observedAt"`
	} `json:"rates"`
}

// Rates asks the provider for the rate from base to quote on the dates of r.
func (c *Client) Rates(ctx context.Context, base, quote string, r Range) ([]Rate, error) {
	for _, code := range []string{base, quote} {
		if !currencyPattern.MatchString(code) {
			return nil, errors.ErrInvalidParameter{Description: fmt.Sprintf(
				"the currency %q is not an ISO 4217 code, such as EUR", code)}
		}
	}
	if base == quote {
		return nil, errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the rate from %s to %s is 1; ask for two different currencies", base, quote)}
	}
	r, err := r.normalised()
	if err != nil {
		return nil, err
	}
	want := r.dates()
	body, err := c.get(ctx, "rates", url.Values{"base": {base}, "quote": {quote}}, r,
		errors.ErrNotFound{ResourceType: "Conversion", ID: base + "/" + quote})
	if err != nil {
		return nil, err
	}

	var wire wireRates
	if err := json.Unmarshal(body, &wire); err != nil {
		return nil, fmt.Errorf("the provider's answer is not rates as the spec defines them: %w", err)
	}
	answered := make([]string, len(wire.Rates))
	for i, w := range wire.Rates {
		answered[i] = w.Date
	}
	dates, err := checkDates(answered, r, want)
	if err != nil {
		return nil, err
	}
	rates := make([]Rate, len(wire.Rates))
	for i, w := range wire.Rates {
		if !ratePattern.MatchString(w.Rate) {
			return nil, fmt.Errorf("the provider answered the rate %q for %s, which breaks the spec: "+
				"it must be a decimal string with at most 18 decimal places", w.Rate, w.Date)
		}
		rate, err := decimal.NewFromString(w.Rate)
		if err != nil {
			return nil, fmt.Errorf("parse the rate %q: %w", w.Rate, err)
		}
		if !rate.IsPositive() {
			return nil, fmt.Errorf("the provider answered a rate of %s for %s, which breaks the spec: "+
				"it must be greater than zero", w.Rate, w.Date)
		}
		if err := checkObserved(dates[i], w.ObservedAt); err != nil {
			return nil, err
		}
		rates[i] = Rate{Date: dates[i], Rate: rate, ObservedAt: w.ObservedAt}
	}
	return rates, nil
}
