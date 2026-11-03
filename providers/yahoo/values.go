package main

import (
	"fmt"
	"net/http"
	"regexp"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
)

// maxValuePlaces is the most decimal places the spec allows a value.
const maxValuePlaces = 8

var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)

// minorUnits maps the minor units Yahoo prices some listings in to their
// currency, a hundred of them to one: London's pence, Johannesburg's cents
// and Tel Aviv's agorot.
var minorUnits = map[string]string{"GBp": "GBP", "GBX": "GBP", "ZAc": "ZAR", "ILA": "ILS"}

type dayValue struct {
	Date       string    `json:"date"`
	Value      string    `json:"value"`
	Currency   string    `json:"currency"`
	ObservedAt time.Time `json:"observedAt"`
}

// values serves GET /v1/values: Yahoo's closes of the asset's symbol.
func (s *server) values(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()
	r, err := parseRange(query)
	if err != nil {
		writeError(w, err)
		return
	}
	if query.Get("isin") == "" && query.Get("ticker") == "" {
		writeError(w, ierrors.ErrMissingRequiredParameter{Parameters: []string{"isin or ticker"}})
		return
	}
	sym, err := symbol(query.Get("ticker"), query.Get("exchange"), query.Get("kind"))
	if err != nil {
		writeError(w, err)
		return
	}
	ch, series, err := s.series(req.Context(), sym, r)
	if err != nil {
		writeError(w, err)
		return
	}

	currency, shift, places := ch.currency, int32(0), ch.places
	if major, ok := minorUnits[currency]; ok {
		currency, shift, places = major, -2, places+2
	}
	if !currencyPattern.MatchString(currency) {
		writeError(w, fmt.Errorf("the Yahoo chart API prices %s in %q, which is not an ISO 4217 code", sym, ch.currency))
		return
	}
	places = min(max(places, 0), maxValuePlaces)

	values := make([]dayValue, len(series))
	for i, d := range series {
		value := d.Row.close.Shift(shift).Round(places)
		if value.IsNegative() {
			writeError(w, fmt.Errorf("the Yahoo chart API answered a close of %s for %s on %s, and the spec allows no negative value",
				value, sym, d.Row.observedAt.Format(time.DateOnly)))
			return
		}
		values[i] = dayValue{
			Date: d.Date.Format(time.DateOnly), Value: value.StringFixed(places), Currency: currency, ObservedAt: d.Row.observedAt,
		}
	}
	writeJSON(w, map[string]any{"values": values})
}
