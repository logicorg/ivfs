package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
)

// lookback is how long before a range Yahoo is asked from, so that a first
// date with no session of its own, such as a Monday after a long weekend, or
// a fund priced once a month, carries the close before it forward.
const lookback = 35 * 24 * time.Hour

// server answers v1 of docs/specs/asset-value-provider.v1.yaml from Yahoo.
type server struct {
	charts *chartClient
}

func newHandler(charts *chartClient) http.Handler {
	s := &server{charts: charts}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /v1/values", s.values)
	mux.HandleFunc("GET /v1/rates", s.rates)
	return mux
}

// parseRange reads the range the spec asks for from query.
func parseRange(query url.Values) (assetvalue.Range, error) {
	var missing []string
	for _, name := range []string{"from", "to"} {
		if query.Get(name) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return assetvalue.Range{}, ierrors.ErrMissingRequiredParameter{Parameters: missing}
	}
	r := assetvalue.Range{Interval: assetvalue.Interval(query.Get("interval"))}
	for _, bound := range []struct {
		name string
		date *time.Time
	}{{"from", &r.From}, {"to", &r.To}} {
		date, err := time.Parse(time.DateOnly, query.Get(bound.name))
		if err != nil {
			return assetvalue.Range{}, ierrors.ErrInvalidParameter{Description: fmt.Sprintf(
				"%s is %q, not a date, such as 2026-10-10", bound.name, query.Get(bound.name))}
		}
		*bound.date = date
	}
	if _, err := r.Dates(); err != nil {
		return assetvalue.Range{}, err
	}
	return r, nil
}

// series returns the bars of symbol answering for the dates of r, which is
// a range parseRange returned, and the chart they are from.
func (s *server) series(ctx context.Context, symbol string, r assetvalue.Range) (chart, []assetvalue.Dated[bar], error) {
	dates, err := r.Dates()
	if err != nil {
		return chart{}, nil, err
	}
	ch, err := s.charts.closes(ctx, symbol, r.From.Add(-lookback), r.To)
	if err != nil {
		return chart{}, nil, err
	}
	series := assetvalue.CarryForward(dates, ch.bars, func(b bar) time.Time { return b.observedAt })
	if len(series) == 0 {
		return chart{}, nil, ierrors.ErrNotFound{ResourceType: "Symbol", ID: symbol}
	}
	return ch, series, nil
}

type problem struct {
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
}

// writeError answers err as a problem: a request the spec does not allow
// is 400, a symbol with no value 404, and anything else Yahoo failing.
func writeError(w http.ResponseWriter, err error) {
	var (
		notFound ierrors.ErrNotFound
		invalid  ierrors.ErrInvalidParameter
		missing  ierrors.ErrMissingRequiredParameter
	)
	status := http.StatusBadGateway
	switch {
	case errors.As(err, &notFound):
		status = http.StatusNotFound
	case errors.As(err, &invalid), errors.As(err, &missing):
		status = http.StatusBadRequest
	default:
		log.Printf("Failed: %v", err)
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Title: http.StatusText(status), Status: status, Detail: err.Error()})
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
