package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// The sessions of a NASDAQ week: each opens at 13:30 UTC and lasts until
// 20:00.
var (
	nasdaqSession = 6*time.Hour + 30*time.Minute
	thursday      = time.Date(2026, 10, 8, 13, 30, 0, 0, time.UTC)
	friday        = time.Date(2026, 10, 9, 13, 30, 0, 0, time.UTC)
	fridayClose   = friday.Add(nasdaqSession)
)

// session is a daily bar as Yahoo writes it: when the session opened, and
// its close, a number as written, or nil for a session with no trades.
type session struct {
	opened time.Time
	close  any
}

// chartJSON is Yahoo's answer for a symbol priced in currency, shown with
// priceHint places, whose sessions last length and whose latest trade was
// at lastTrade.
func chartJSON(currency string, priceHint int, length time.Duration, lastTrade time.Time, sessions ...session) string {
	timestamps := make([]int64, len(sessions))
	closes := make([]any, len(sessions))
	for i, s := range sessions {
		timestamps[i] = s.opened.Unix()
		if s.close != nil {
			closes[i] = json.RawMessage(s.close.(string))
		}
	}
	body, err := json.Marshal(map[string]any{"chart": map[string]any{"error": nil, "result": []any{map[string]any{
		"meta": map[string]any{
			"currency": currency, "priceHint": priceHint, "regularMarketTime": lastTrade.Unix(),
			"currentTradingPeriod": map[string]any{"regular": map[string]any{
				"start": 1_000_000, "end": 1_000_000 + int64(length/time.Second),
			}},
		},
		"timestamp":  timestamps,
		"indicators": map[string]any{"quote": []any{map[string]any{"close": closes}}},
	}}}})
	if err != nil {
		panic(err)
	}
	return string(body)
}

// notFoundJSON is Yahoo's answer for a symbol it does not list.
const notFoundJSON = `{"chart":{"result":null,"error":{"code":"Not Found","description":"No data found, symbol may be delisted"}}}`

// answer is what the fake Yahoo answers.
type answer struct {
	status int
	body   string
}

// fakeYahoo serves a for symbol's daily bars from from to to, both
// included, and fails t on any other request, counting them all. With no
// symbol, every request fails t.
func fakeYahoo(t *testing.T, symbol string, from, to time.Time, a answer) (*url.URL, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		requests.Add(1)
		want := url.Values{
			"period1":  {strconv.FormatInt(from.Unix(), 10)},
			"period2":  {strconv.FormatInt(to.AddDate(0, 0, 1).Unix(), 10)},
			"interval": {"1d"},
		}
		if symbol == "" || req.URL.Path != "/v8/finance/chart/"+symbol || req.URL.Query().Encode() != want.Encode() ||
			req.Header.Get("User-Agent") == "" {
			t.Errorf("asked for %s?%s, not %s?%s", req.URL.Path, req.URL.RawQuery, symbol, want.Encode())
			w.WriteHeader(http.StatusTeapot)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(a.status)
		fmt.Fprint(w, a.body)
	}))
	t.Cleanup(server.Close)
	baseURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	return baseURL, &requests
}

func TestChartClientCloses(t *testing.T) {
	from, to := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	wednesday := time.Date(2026, 10, 7, 13, 30, 0, 0, time.UTC)
	tests := map[string]struct {
		answer  answer
		want    chart
		wantErr error
		// wantFailure is the message of an error that is not a domain one.
		wantFailure string
	}{
		"each close when its session ends, one with no trades left out": {
			answer: answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, fridayClose.Add(time.Second),
				session{wednesday, nil}, session{thursday, "229.10000610351562"}, session{friday, "231.4499969482422"})},
			want: chart{currency: "USD", places: 2, bars: []bar{
				{close: decimal.RequireFromString("229.10000610351562"), observedAt: thursday.Add(nasdaqSession)},
				{close: decimal.RequireFromString("231.4499969482422"), observedAt: fridayClose},
			}},
		},
		"a session still open, at its latest trade": {
			answer: answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, friday.Add(time.Hour),
				session{thursday, "229.1"}, session{friday, "230.5"})},
			want: chart{currency: "USD", places: 2, bars: []bar{
				{close: decimal.RequireFromString("229.1"), observedAt: thursday.Add(nasdaqSession)},
				{close: decimal.RequireFromString("230.5"), observedAt: friday.Add(time.Hour)},
			}},
		},
		"no sessions": {
			answer: answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, fridayClose)},
			want:   chart{currency: "USD", places: 2},
		},
		"a symbol Yahoo does not list": {
			answer:  answer{http.StatusNotFound, notFoundJSON},
			wantErr: ierrors.ErrNotFound{ResourceType: "Symbol", ID: "AAPL"},
		},
		"a symbol Yahoo does not list, answered as a chart": {
			answer:  answer{http.StatusOK, notFoundJSON},
			wantErr: ierrors.ErrNotFound{ResourceType: "Symbol", ID: "AAPL"},
		},
		"rate limited": {
			answer:      answer{http.StatusTooManyRequests, "Too Many Requests"},
			wantFailure: "the Yahoo chart API is rate limiting this address; try again in a few minutes",
		},
		"a failure Yahoo describes": {
			answer: answer{http.StatusBadRequest,
				`{"chart":{"result":null,"error":{"code":"Bad Request","description":"Invalid input - start date cannot be after end date"}}}`},
			wantFailure: "the Yahoo chart API answered 400 Bad Request for AAPL: Invalid input - start date cannot be after end date",
		},
		"a failure it does not": {
			answer:      answer{http.StatusServiceUnavailable, "<html>down</html>"},
			wantFailure: "the Yahoo chart API answered 503 Service Unavailable for AAPL",
		},
		"not a chart": {
			answer:      answer{http.StatusOK, "<html>consent</html>"},
			wantFailure: "the Yahoo chart API's answer for AAPL is not a chart: invalid character '<' looking for beginning of value",
		},
		"fewer closes than sessions": {
			answer: answer{http.StatusOK, `{"chart":{"result":[{"meta":{"currency":"USD"},"timestamp":[1],` +
				`"indicators":{"quote":[{"close":[]}]}}],"error":null}}`},
			wantFailure: "the Yahoo chart API answered 1 sessions and 0 closes for AAPL",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			baseURL, _ := fakeYahoo(t, "AAPL", from, to, tt.answer)
			got, err := newChartClient(baseURL, http.DefaultClient).closes(context.Background(), "AAPL", from, to)
			switch {
			case tt.wantErr != nil:
				require.Equal(t, tt.wantErr, err)
			case tt.wantFailure != "":
				require.EqualError(t, err, tt.wantFailure)
			default:
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

// TestChartClientCache checks that an answer, or a symbol Yahoo does not
// list, is reused until it expires, and a failure is not.
func TestChartClientCache(t *testing.T) {
	from, to := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	closes := answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, fridayClose, session{friday, "231.45"})}
	tests := map[string]struct {
		answer answer
		// after is how long after the first request the second one is.
		after        time.Duration
		wantRequests int32
	}{
		"closes, reused":                   {answer: closes, after: cacheTTL - time.Second, wantRequests: 1},
		"closes, asked again once expired": {answer: closes, after: cacheTTL, wantRequests: 2},
		"a symbol Yahoo does not list, reused": {
			answer: answer{http.StatusNotFound, notFoundJSON}, after: time.Minute, wantRequests: 1,
		},
		"a failure, asked again": {
			answer: answer{http.StatusServiceUnavailable, ""}, after: time.Second, wantRequests: 2,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			baseURL, requests := fakeYahoo(t, "AAPL", from, to, tt.answer)
			charts := newChartClient(baseURL, http.DefaultClient)
			now := time.Date(2026, 10, 9, 21, 0, 0, 0, time.UTC)
			charts.now = func() time.Time { return now }

			first, firstErr := charts.closes(context.Background(), "AAPL", from, to)
			now = now.Add(tt.after)
			second, secondErr := charts.closes(context.Background(), "AAPL", from, to)
			require.Equal(t, tt.wantRequests, requests.Load())
			require.Equal(t, first, second)
			require.Equal(t, firstErr, secondErr)
		})
	}
}
