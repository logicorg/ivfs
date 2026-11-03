package main

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRates(t *testing.T) {
	day := func(d int) time.Time { return time.Date(2026, 10, d, 0, 0, 0, 0, time.UTC) }
	// A currency's sessions run from midnight in London, 23:00 UTC in
	// summer, for a day less a minute.
	fx := 23*time.Hour + 59*time.Minute
	fxThursday, fxFriday := day(7).Add(23*time.Hour), day(8).Add(23*time.Hour)
	pair := func(currency, thursdayClose, fridayClose string) answer {
		return answer{http.StatusOK, chartJSON(currency, 4, fx, fxFriday.Add(fx),
			session{fxThursday, thursdayClose}, session{fxFriday, fridayClose})}
	}
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
		"EUR in dollars, over a weekend": {
			path:   "/v1/rates?base=EUR&quote=USD&from=2026-10-08&to=2026-10-10",
			symbol: "EURUSD=X", from: day(8).Add(-lookback), to: day(10),
			answer:     pair("USD", "1.0838000774383545", "1.0842000246047974"),
			wantStatus: http.StatusOK,
			wantBody: `{"rates":[{"date":"2026-10-08","rate":"1.0838","observedAt":"2026-10-08T22:59:00Z"},` +
				`{"date":"2026-10-09","rate":"1.0842","observedAt":"2026-10-09T22:59:00Z"},` +
				`{"date":"2026-10-10","rate":"1.0842","observedAt":"2026-10-09T22:59:00Z"}]}`,
		},
		"a small rate, to six significant digits": {
			path:   "/v1/rates?base=JPY&quote=EUR&from=2026-10-09&to=2026-10-09",
			symbol: "JPYEUR=X", from: day(9).Add(-lookback), to: day(9),
			answer:     pair("EUR", "0.006130000110715628", "0.006134567037224770"),
			wantStatus: http.StatusOK,
			wantBody:   `{"rates":[{"date":"2026-10-09","rate":"0.00613457","observedAt":"2026-10-09T22:59:00Z"}]}`,
		},
		"a large rate, to six significant digits": {
			path:   "/v1/rates?base=USD&quote=JPY&from=2026-10-09&to=2026-10-09",
			symbol: "USDJPY=X", from: day(9).Add(-lookback), to: day(9),
			answer:     pair("JPY", "149.1", "149.23500061035156"),
			wantStatus: http.StatusOK,
			wantBody:   `{"rates":[{"date":"2026-10-09","rate":"149.235","observedAt":"2026-10-09T22:59:00Z"}]}`,
		},
		"a pair Yahoo does not list": {
			path:   "/v1/rates?base=EUR&quote=XAF&from=2026-10-09&to=2026-10-09",
			symbol: "EURXAF=X", from: day(9).Add(-lookback), to: day(9), answer: answer{http.StatusNotFound, notFoundJSON},
			wantStatus: http.StatusNotFound,
			wantBody:   `{"title":"Not Found","status":404,"detail":"resource Symbol not found with ID: EURXAF=X"}`,
		},
		"a rate in another currency than the quote": {
			path:   "/v1/rates?base=EUR&quote=USD&from=2026-10-09&to=2026-10-09",
			symbol: "EURUSD=X", from: day(9).Add(-lookback), to: day(9),
			answer:     pair("GBP", "0.86", "0.87"),
			wantStatus: http.StatusBadGateway,
			wantBody:   `{"title":"Bad Gateway","status":502,"detail":"the Yahoo chart API prices EURUSD=X in GBP, not USD"}`,
		},
		"a rate of zero": {
			path:   "/v1/rates?base=EUR&quote=USD&from=2026-10-09&to=2026-10-09",
			symbol: "EURUSD=X", from: day(9).Add(-lookback), to: day(9),
			answer:     pair("USD", "1.08", "0"),
			wantStatus: http.StatusBadGateway,
			wantBody: `{"title":"Bad Gateway","status":502,"detail":"the Yahoo chart API answered a rate of 0 ` +
				`for EURUSD=X on 2026-10-09, and the spec allows only one above zero"}`,
		},
		"the same currency": {
			path:       "/v1/rates?base=EUR&quote=EUR&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,` +
				`"detail":"invalid parameter: the rate from EUR to EUR is 1; ask for two different currencies"}`,
		},
		"codes that are not ones, the base reported": {
			path:       "/v1/rates?base=eur&quote=dollar&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,` +
				`"detail":"invalid parameter: base is \"eur\", not an ISO 4217 code, such as EUR"}`,
		},
		"no quote": {
			path:       "/v1/rates?base=EUR&from=2026-10-09&to=2026-10-09",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,` +
				`"detail":"invalid parameter: quote is \"\", not an ISO 4217 code, such as EUR"}`,
		},
		"an interval that is not one": {
			path:       "/v1/rates?base=EUR&quote=USD&from=2026-10-09&to=2026-10-09&interval=year",
			wantStatus: http.StatusBadRequest,
			wantBody: `{"title":"Bad Request","status":400,` +
				`"detail":"invalid parameter: the interval \"year\" is not one of day, week and month"}`,
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
