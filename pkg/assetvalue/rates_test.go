package assetvalue

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// dayRate is one rate as a provider puts it on the wire.
func dayRate(date, rate, observedAt string) string {
	return fmt.Sprintf(`{"date": %q, "rate": %q, "observedAt": %q}`, date, rate, observedAt)
}

func ratesBody(days ...string) string {
	return `{"rates": [` + strings.Join(days, ", ") + `]}`
}

func TestClientRates(t *testing.T) {
	friday, saturday := date(2026, 10, 9), date(2026, 10, 10)
	const fixedAt = "2026-10-09T16:00:00Z"
	fridayFixing := Rate{
		Date: friday, Rate: decimal.RequireFromString("1.0921"), ObservedAt: time.Date(2026, 10, 9, 16, 0, 0, 0, time.UTC),
	}
	carried := func(date time.Time) Rate {
		r := fridayFixing
		r.Date = date
		return r
	}
	twoDays := Range{From: friday, To: saturday}
	oneDay := Range{From: friday, To: friday}

	tests := map[string]struct {
		base, quote string
		r           Range
		// status, contentType, and body are the provider's answer.
		status      int
		contentType string
		body        string
		wantQuery   url.Values
		want        []Rate
		wantErr     string
		wantNoValue bool
		// wantNoCall is set when the client answers without asking.
		wantNoCall bool
	}{
		"a rate carried over to Saturday": {
			base: "EUR", quote: "USD", r: twoDays, status: http.StatusOK,
			body: ratesBody(dayRate("2026-10-09", "1.0921", fixedAt), dayRate("2026-10-10", "1.0921", fixedAt)),
			wantQuery: url.Values{
				"base": {"EUR"}, "quote": {"USD"}, "from": {"2026-10-09"}, "to": {"2026-10-10"}, "interval": {"day"},
			},
			want: []Rate{fridayFixing, carried(saturday)},
		},
		"every week, with all 18 decimal places": {
			base: "BTC", quote: "JPY", r: Range{From: date(2026, 10, 5), To: date(2026, 10, 14), Interval: Weekly},
			status: http.StatusOK,
			body: ratesBody(
				dayRate("2026-10-11", "0.000000061234567891", "2026-10-11T23:00:00Z"),
				dayRate("2026-10-14", "0.000000062", "2026-10-14T23:00:00Z"),
			),
			wantQuery: url.Values{
				"base": {"BTC"}, "quote": {"JPY"}, "from": {"2026-10-05"}, "to": {"2026-10-14"}, "interval": {"week"},
			},
			want: []Rate{
				{Date: date(2026, 10, 11), Rate: decimal.RequireFromString("0.000000061234567891"), ObservedAt: time.Date(2026, 10, 11, 23, 0, 0, 0, time.UTC)},
				{Date: date(2026, 10, 14), Rate: decimal.RequireFromString("0.000000062"), ObservedAt: time.Date(2026, 10, 14, 23, 0, 0, 0, time.UTC)},
			},
		},
		"no rate, or no rates at all": {
			base: "EUR", quote: "USD", r: twoDays, status: http.StatusNotFound, body: "404 page not found",
			wantNoValue: true,
		},
		"a currency that is not an ISO 4217 code, without calling the provider": {
			base: "eur", quote: "USD", r: twoDays,
			wantErr: `the currency "eur" is not an ISO 4217 code, such as EUR`, wantNoCall: true,
		},
		"a pair of the same currency, without calling the provider": {
			base: "EUR", quote: "EUR", r: twoDays,
			wantErr: "the rate from EUR to EUR is 1; ask for two different currencies", wantNoCall: true,
		},
		"a range that ends before it starts, without calling the provider": {
			base: "EUR", quote: "USD", r: Range{From: saturday, To: friday},
			wantErr: "the range ends on 2026-10-09, before it starts on 2026-10-10", wantNoCall: true,
		},
		"a failure": {
			base: "EUR", quote: "USD", r: twoDays, status: http.StatusServiceUnavailable, contentType: "application/problem+json",
			body:    `{"title": "Service Unavailable", "detail": "the ECB feed is down"}`,
			wantErr: "the provider answered 503 Service Unavailable: the ECB feed is down",
		},
		"an answer that is not rates": {
			base: "EUR", quote: "USD", r: oneDay, status: http.StatusOK,
			body:    ratesBody(`{"date": "2026-10-09", "rate": 1.0921, "observedAt": "2026-10-09T16:00:00Z"}`),
			wantErr: "the provider's answer is not rates as the spec defines them",
		},
		"no rates": {
			base: "EUR", quote: "USD", r: twoDays, status: http.StatusOK, body: `{"rates": []}`,
			wantErr: "the provider answered nothing, which breaks the spec: it answers 404 instead",
		},
		"rates that stop before the range ends": {
			base: "EUR", quote: "USD", r: twoDays, status: http.StatusOK,
			body:    ratesBody(dayRate("2026-10-09", "1.0921", fixedAt)),
			wantErr: "the provider stops on 2026-10-09, which breaks the spec: it goes on to 2026-10-10",
		},
		"a rate of zero": {
			base: "EUR", quote: "USD", r: oneDay, status: http.StatusOK,
			body:    ratesBody(dayRate("2026-10-09", "0.000", fixedAt)),
			wantErr: "the provider answered a rate of 0.000 for 2026-10-09, which breaks the spec: it must be greater than zero",
		},
		"a rate with more than 18 decimal places": {
			base: "EUR", quote: "USD", r: oneDay, status: http.StatusOK,
			body:    ratesBody(dayRate("2026-10-09", "1.0000000000000000001", fixedAt)),
			wantErr: `the provider answered the rate "1.0000000000000000001" for 2026-10-09, which breaks the spec`,
		},
		"a rate observed after its day ended": {
			base: "EUR", quote: "USD", r: oneDay, status: http.StatusOK,
			body:    ratesBody(dayRate("2026-10-09", "1.0921", "2026-10-10T08:00:00Z")),
			wantErr: "the provider answered a value for 2026-10-09 observed at 2026-10-10T08:00:00Z, after the day ended",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, received := provider(t, tc.status, tc.contentType, tc.body)

			got, err := client.Rates(context.Background(), tc.base, tc.quote, tc.r)

			switch {
			case tc.wantNoValue:
				var notFound errors.ErrNotFound
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, "Conversion", notFound.ResourceType)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
				require.Len(t, got, len(tc.want))
				for i, want := range tc.want {
					require.True(t, want.Date.Equal(got[i].Date), "date %d is %s, want %s", i, got[i].Date, want.Date)
					require.True(t, want.Rate.Equal(got[i].Rate), "rate %s, want %s", got[i].Rate, want.Rate)
					require.True(t, want.ObservedAt.Equal(got[i].ObservedAt))
				}
			}
			if tc.wantNoCall {
				require.Nil(t, received())
				return
			}
			require.NotNil(t, received())
			require.Equal(t, "/prefix/v1/rates", received().URL.Path)
			if tc.wantQuery != nil {
				require.Equal(t, tc.wantQuery, received().URL.Query())
			}
		})
	}
}
