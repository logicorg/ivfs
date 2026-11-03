package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// get asks a provider answering from the chart API at baseURL for path, and
// returns the status and body of its answer.
func get(t *testing.T, charts *chartClient, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	newHandler(charts).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, err := io.ReadAll(rec.Body)
	require.NoError(t, err)
	return rec.Code, string(body)
}

func TestValues(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	week := answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, fridayClose.Add(time.Second),
		session{thursday, "229.10000610351562"}, session{friday, "231.4499969482422"})}
	fridayValue := `{"date":"%s","value":"231.45","currency":"USD","observedAt":"2026-10-09T20:00:00Z"}`
	tests := map[string]struct {
		path string
		// symbol is what Yahoo is asked for, over the sessions from from to
		// to, or empty when it is not asked.
		symbol     string
		from, to   time.Time
		answer     answer
		wantStatus int
		wantBody   string
	}{
		"a weekend, which repeats Friday's close": {
			path:   "/v1/values?ticker=AAPL&exchange=NASDAQ&kind=stock&from=2026-10-09&to=2026-10-11",
			symbol: "AAPL", from: day(9).Add(-lookback), to: day(11), answer: week,
			wantStatus: http.StatusOK,
			wantBody: `{"values":[` + fmt.Sprintf(fridayValue, "2026-10-09") + `,` + fmt.Sprintf(fridayValue, "2026-10-10") + `,` +
				fmt.Sprintf(fridayValue, "2026-10-11") + `]}`,
		},
		"a range starting on a Saturday, from the close before it": {
			path:   "/v1/values?ticker=AAPL&from=2026-10-10&to=2026-10-10",
			symbol: "AAPL", from: day(10).Add(-lookback), to: day(10), answer: week,
			wantStatus: http.StatusOK, wantBody: `{"values":[` + fmt.Sprintf(fridayValue, "2026-10-10") + `]}`,
		},
		"every Sunday, starting with the first that has a close": {
			path:   "/v1/values?ticker=AAPL&from=2026-10-01&to=2026-10-14&interval=week",
			symbol: "AAPL", from: day(1).Add(-lookback), to: day(14), answer: week,
			wantStatus: http.StatusOK,
			wantBody:   `{"values":[` + fmt.Sprintf(fridayValue, "2026-10-11") + `,` + fmt.Sprintf(fridayValue, "2026-10-14") + `]}`,
		},
		"pence, in pounds": {
			path:   "/v1/values?ticker=VWRL&exchange=LSE&from=2026-10-09&to=2026-10-09",
			symbol: "VWRL.L", from: day(9).Add(-lookback), to: day(9),
			answer: answer{http.StatusOK, chartJSON("GBp", 2, 8*time.Hour+30*time.Minute, day(9).Add(17*time.Hour),
				session{day(9).Add(7 * time.Hour), "10234.5"})},
			wantStatus: http.StatusOK,
			wantBody:   `{"values":[{"date":"2026-10-09","value":"102.3450","currency":"GBP","observedAt":"2026-10-09T15:30:00Z"}]}`,
		},
		"no session in the range": {
			path:   "/v1/values?ticker=AAPL&from=2026-10-09&to=2026-10-09",
			symbol: "AAPL", from: day(9).Add(-lookback), to: day(9),
			answer:     answer{http.StatusOK, chartJSON("USD", 2, nasdaqSession, fridayClose)},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"resource Symbol not found with ID: AAPL"}`,
		},
		"a symbol Yahoo does not list": {
			path:   "/v1/values?ticker=NOPE&from=2026-10-09&to=2026-10-09",
			symbol: "NOPE", from: day(9).Add(-lookback), to: day(9), answer: answer{http.StatusNotFound, notFoundJSON},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"resource Symbol not found with ID: NOPE"}`,
		},
		"an ISIN alone, which gives no symbol": {
			path:       "/v1/values?isin=IE00BK5BQT80&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"resource Symbol not found with ID: none, as the asset has no ticker"}`,
		},
		"Yahoo failing": {
			path:   "/v1/values?ticker=AAPL&from=2026-10-09&to=2026-10-09",
			symbol: "AAPL", from: day(9).Add(-lookback), to: day(9), answer: answer{http.StatusTooManyRequests, ""},
			wantStatus: http.StatusBadGateway,
			wantBody: `{"title":"Bad Gateway","status":502,` +
				`"detail":"the Yahoo chart API is rate limiting this address; try again in a few minutes"}`,
		},
		"a currency that is not a code": {
			path:   "/v1/values?ticker=AAPL&from=2026-10-09&to=2026-10-09",
			symbol: "AAPL", from: day(9).Add(-lookback), to: day(9),
			answer:     answer{http.StatusOK, chartJSON("usd", 2, nasdaqSession, fridayClose, session{friday, "231.45"})},
			wantStatus: http.StatusBadGateway,
			wantBody: `{"title":"Bad Gateway","status":502,` +
				`"detail":"the Yahoo chart API prices AAPL in \"usd\", which is not an ISO 4217 code"}`,
		},
		"neither an ISIN nor a ticker": {
			path:       "/v1/values?exchange=XETRA&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody:   `{"title":"Bad Request","status":400,"detail":"missing required parameters: isin or ticker"}`,
		},
		"an exchange it does not know": {
			path:       "/v1/values?ticker=VWCE&exchange=Nowhere&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,"detail":"invalid parameter: the exchange \"NOWHERE\" is not one ` +
				`the Yahoo provider knows; enter its name or MIC, such as NASDAQ, XETRA, LSE or XMIL, ` +
				`or leave it empty to look the ticker up as it is"}`,
		},
		"a range that ends before it starts": {
			path:       "/v1/values?ticker=AAPL&from=2026-10-10&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,` +
				`"detail":"invalid parameter: the range ends on 2026-10-09, before it starts on 2026-10-10"}`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			baseURL, _ := fakeYahoo(t, tt.symbol, tt.from, tt.to, tt.answer)
			status, body := get(t, newChartClient(baseURL, http.DefaultClient), tt.path)
			require.Equal(t, tt.wantStatus, status)
			require.JSONEq(t, tt.wantBody, body)
		})
	}
}
