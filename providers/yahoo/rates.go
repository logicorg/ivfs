package main

import (
	"fmt"
	"math"
	"net/http"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// rateDigits is how many significant digits a rate keeps: Yahoo's are
// single-precision floats, good to about seven, and a rate as small as
// JPY's in EUR needs them past the fourth decimal place.
const rateDigits = 6

// maxRatePlaces is the most decimal places the spec allows a rate.
const maxRatePlaces = 18

type dayRate struct {
	Date       string    `json:"date"`
	Rate       string    `json:"rate"`
	ObservedAt time.Time `json:"observedAt"`
}

// rates serves GET /v1/rates: Yahoo's closes of the pair's symbol, such as
// EURUSD=X.
func (s *server) rates(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	base, quote := query.Get("base"), query.Get("quote")
	for _, param := range []string{"base", "quote"} {
		if code := query.Get(param); !currencyPattern.MatchString(code) {
			writeError(w, ierrors.ErrInvalidParameter{Description: fmt.Sprintf(
				"%s is %q, not an ISO 4217 code, such as EUR", param, code)})
			return
		}
	}
	if base == quote {
		writeError(w, ierrors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the rate from %s to %s is 1; ask for two different currencies", base, quote)})
		return
	}
	r, err := parseRange(query)
	if err != nil {
		writeError(w, err)
		return
	}
	sym := base + quote + "=X"
	ch, series, err := s.series(req.Context(), sym, r)
	if err != nil {
		writeError(w, err)
		return
	}
	if ch.currency != quote {
		writeError(w, fmt.Errorf("the Yahoo chart API prices %s in %s, not %s", sym, ch.currency, quote))
		return
	}

	rates := make([]dayRate, len(series))
	for i, d := range series {
		if !d.Row.close.IsPositive() {
			writeError(w, fmt.Errorf("the Yahoo chart API answered a rate of %s for %s on %s, and the spec allows only one above zero",
				d.Row.close, sym, d.Row.observedAt.Format(time.DateOnly)))
			return
		}
		rates[i] = dayRate{Date: d.Date.Format(time.DateOnly), Rate: significant(d.Row.close), ObservedAt: d.Row.observedAt}
	}
	writeJSON(w, map[string]any{"rates": rates})
}

// significant returns rate, which is positive, to rateDigits significant
// digits, without trailing zeros.
func significant(rate decimal.Decimal) string {
	// The position of the leading digit: 0 for 1.08, -3 for 0.00613.
	lead := int32(math.Floor(math.Log10(rate.InexactFloat64())))
	places := min(max(rateDigits-1-lead, 0), maxRatePlaces)
	return rate.Round(places).String()
}
