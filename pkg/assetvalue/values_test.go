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
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// dayValue is one value as a provider puts it on the wire.
func dayValue(date, value, currency, observedAt string) string {
	return fmt.Sprintf(`{"date": %q, "value": %q, "currency": %q, "observedAt": %q}`, date, value, currency, observedAt)
}

func valuesBody(days ...string) string {
	return `{"values": [` + strings.Join(days, ", ") + `]}`
}

func TestClientValues(t *testing.T) {
	friday, saturday, sunday := date(2026, 10, 9), date(2026, 10, 10), date(2026, 10, 11)
	apple := model.Asset{
		Name: "Apple", Kind: model.AssetKindStock,
		ISIN: new("US0378331005"), Ticker: new("AAPL"), Exchange: new("NASDAQ"),
	}
	const closedAt = "2026-10-09T20:00:00Z"
	fridayClose := Value{
		Date: friday, Amount: decimal.RequireFromString("231.45"), Currency: "USD",
		ObservedAt: time.Date(2026, 10, 9, 20, 0, 0, 0, time.UTC),
	}
	// carried is the Friday close as the value on a later date.
	carried := func(date time.Time) Value {
		v := fridayClose
		v.Date = date
		return v
	}
	weekend := Range{From: friday, To: sunday}

	tests := map[string]struct {
		asset model.Asset
		r     Range
		// status, contentType, and body are the provider's answer.
		status      int
		contentType string
		body        string
		wantQuery   url.Values
		want        []Value
		wantErr     string
		wantNoValue bool
		// wantNoCall is set when the client answers without asking.
		wantNoCall bool
	}{
		"every identifier, over a weekend": {
			asset: apple, r: weekend, status: http.StatusOK,
			body: valuesBody(
				dayValue("2026-10-09", "231.45", "USD", closedAt),
				dayValue("2026-10-10", "231.45", "USD", closedAt),
				dayValue("2026-10-11", "231.45", "USD", closedAt),
			),
			wantQuery: url.Values{
				"isin": {"US0378331005"}, "ticker": {"AAPL"}, "exchange": {"NASDAQ"}, "kind": {"stock"},
				"from": {"2026-10-09"}, "to": {"2026-10-11"}, "interval": {"day"},
			},
			want: []Value{fridayClose, carried(saturday), carried(sunday)},
		},
		"a single day": {
			asset: apple, r: Range{From: saturday, To: saturday}, status: http.StatusOK,
			body: valuesBody(dayValue("2026-10-10", "231.45", "USD", closedAt)),
			want: []Value{carried(saturday)},
		},
		"every month, then the day the range ends on": {
			asset: apple, r: Range{From: date(2026, 8, 15), To: saturday, Interval: Monthly}, status: http.StatusOK,
			body: valuesBody(
				dayValue("2026-08-31", "220", "USD", "2026-08-31T20:00:00Z"),
				dayValue("2026-09-30", "225.5", "USD", "2026-09-30T20:00:00Z"),
				dayValue("2026-10-10", "231.45", "USD", closedAt),
			),
			wantQuery: url.Values{
				"isin": {"US0378331005"}, "ticker": {"AAPL"}, "exchange": {"NASDAQ"}, "kind": {"stock"},
				"from": {"2026-08-15"}, "to": {"2026-10-10"}, "interval": {"month"},
			},
			want: []Value{
				{Date: date(2026, 8, 31), Amount: decimal.RequireFromString("220"), Currency: "USD", ObservedAt: time.Date(2026, 8, 31, 20, 0, 0, 0, time.UTC)},
				{Date: date(2026, 9, 30), Amount: decimal.RequireFromString("225.5"), Currency: "USD", ObservedAt: time.Date(2026, 9, 30, 20, 0, 0, 0, time.UTC)},
				carried(saturday),
			},
		},
		"values from the first date the provider has one": {
			asset: apple, r: Range{From: friday.AddDate(0, 0, -7), To: saturday}, status: http.StatusOK,
			body: valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt), dayValue("2026-10-10", "231.45", "USD", closedAt)),
			want: []Value{fridayClose, carried(saturday)},
		},
		"an ISIN alone, with blank identifiers left out": {
			asset: model.Asset{Name: "Fund", Kind: model.AssetKindFund, ISIN: new("IE00BK5BQT80"), Ticker: new(" ")},
			r:     Range{From: friday, To: friday}, status: http.StatusOK,
			body: valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt)),
			wantQuery: url.Values{
				"isin": {"IE00BK5BQT80"}, "kind": {"fund"}, "from": {"2026-10-09"}, "to": {"2026-10-09"}, "interval": {"day"},
			},
			want: []Value{fridayClose},
		},
		"no value": {
			asset: apple, r: weekend, status: http.StatusNotFound, contentType: "application/problem+json",
			body:        `{"title": "Not Found", "status": 404}`,
			wantNoValue: true,
		},
		"no identifier the spec knows, without calling the provider": {
			asset: model.Asset{Name: "House", Kind: model.AssetKindOther}, r: weekend,
			wantNoValue: true, wantNoCall: true,
		},
		"a range that ends before it starts, without calling the provider": {
			asset: apple, r: Range{From: sunday, To: friday},
			wantErr: "the range ends on 2026-10-09, before it starts on 2026-10-11", wantNoCall: true,
		},
		"an interval the spec does not have, without calling the provider": {
			asset: apple, r: Range{From: friday, To: sunday, Interval: "hour"},
			wantErr: `the interval "hour" is not one of day, week and month`, wantNoCall: true,
		},
		"a failure, with its problem's detail": {
			asset: apple, r: weekend, status: http.StatusServiceUnavailable, contentType: "application/problem+json",
			body:    `{"title": "Service Unavailable", "detail": "Yahoo rate limited the provider"}`,
			wantErr: "the provider answered 503 Service Unavailable: Yahoo rate limited the provider",
		},
		"a failure, with its problem's title alone": {
			asset: apple, r: weekend, status: http.StatusBadRequest, contentType: "application/problem+json",
			body:    `{"title": "Bad Request"}`,
			wantErr: "the provider answered 400 Bad Request: Bad Request",
		},
		"a failure that is not a problem": {
			asset: apple, r: weekend, status: http.StatusBadGateway, contentType: "text/html",
			body:    "<h1>Bad gateway</h1>",
			wantErr: "the provider answered 502 Bad Gateway",
		},
		"an answer that is not JSON": {
			asset: apple, r: weekend, status: http.StatusOK, body: "<html></html>",
			wantErr: "the provider's answer is not values as the spec defines them",
		},
		"no values": {
			asset: apple, r: weekend, status: http.StatusOK, body: `{"values": []}`,
			wantErr: "the provider answered nothing, which breaks the spec: it answers 404 instead",
		},
		"a date that is not a date": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09T00:00:00Z", "231.45", "USD", closedAt)),
			wantErr: `the provider answered the date "2026-10-09T00:00:00Z", which breaks the spec`,
		},
		"a date before the range": {
			asset: apple, r: Range{From: saturday, To: saturday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt), dayValue("2026-10-10", "231.45", "USD", closedAt)),
			wantErr: "the provider answered 2026-10-09, outside the range from 2026-10-10 to 2026-10-10",
		},
		"a date after the range": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt), dayValue("2026-10-10", "231.45", "USD", closedAt)),
			wantErr: "the provider answered 2026-10-10, outside the range from 2026-10-09 to 2026-10-09",
		},
		"a date that does not end a week": {
			asset: apple, r: Range{From: date(2026, 10, 5), To: date(2026, 10, 21), Interval: Weekly}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt)),
			wantErr: "the provider answered 2026-10-09, which breaks the spec: it is neither the last day of a week nor the end of the range",
		},
		"a missing day": {
			asset: apple, r: weekend, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt), dayValue("2026-10-11", "231.45", "USD", closedAt)),
			wantErr: "the provider answered 2026-10-11 after 2026-10-09, which breaks the spec: it answers one value a day, oldest first, with none missing",
		},
		"days newest first": {
			asset: apple, r: Range{From: friday, To: saturday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-10", "231.45", "USD", closedAt), dayValue("2026-10-09", "231.45", "USD", closedAt)),
			wantErr: "the provider answered 2026-10-09 after 2026-10-10",
		},
		"values that stop before the range ends": {
			asset: apple, r: weekend, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", closedAt)),
			wantErr: "the provider stops on 2026-10-09, which breaks the spec: it goes on to 2026-10-11",
		},
		"a value as a JSON number": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(`{"date": "2026-10-09", "value": 231.45, "currency": "USD", "observedAt": "2026-10-09T20:00:00Z"}`),
			wantErr: "the provider's answer is not values as the spec defines them",
		},
		"a negative value": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "-1", "USD", closedAt)),
			wantErr: `the provider answered the value "-1" for 2026-10-09, which breaks the spec`,
		},
		"a value with more than 8 decimal places": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "0.123456789", "USD", closedAt)),
			wantErr: `the provider answered the value "0.123456789" for 2026-10-09, which breaks the spec`,
		},
		"a currency that is not an ISO 4217 code": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "usd", closedAt)),
			wantErr: `the provider answered the currency "usd" for 2026-10-09, which breaks the spec`,
		},
		"no observedAt": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(`{"date": "2026-10-09", "value": "231.45", "currency": "USD"}`),
			wantErr: "the provider answered no observedAt for 2026-10-09",
		},
		"a value observed after its day ended": {
			asset: apple, r: Range{From: friday, To: friday}, status: http.StatusOK,
			body:    valuesBody(dayValue("2026-10-09", "231.45", "USD", "2026-10-10T00:00:00Z")),
			wantErr: "the provider answered a value for 2026-10-09 observed at 2026-10-10T00:00:00Z, after the day ended",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, received := provider(t, tc.status, tc.contentType, tc.body)

			got, err := client.Values(context.Background(), tc.asset, tc.r)

			switch {
			case tc.wantNoValue:
				var notFound errors.ErrNotFound
				require.ErrorAs(t, err, &notFound)
				require.Equal(t, "AssetValue", notFound.ResourceType)
			case tc.wantErr != "":
				require.ErrorContains(t, err, tc.wantErr)
			default:
				require.NoError(t, err)
				require.Len(t, got, len(tc.want))
				for i, want := range tc.want {
					require.True(t, want.Date.Equal(got[i].Date), "date %d is %s, want %s", i, got[i].Date, want.Date)
					require.True(t, want.Amount.Equal(got[i].Amount), "amount %s, want %s", got[i].Amount, want.Amount)
					require.Equal(t, want.Currency, got[i].Currency)
					require.True(t, want.ObservedAt.Equal(got[i].ObservedAt))
				}
			}
			if tc.wantNoCall {
				require.Nil(t, received())
				return
			}
			require.NotNil(t, received())
			require.Equal(t, http.MethodGet, received().Method)
			require.Equal(t, "/prefix/v1/values", received().URL.Path)
			if tc.wantQuery != nil {
				require.Equal(t, tc.wantQuery, received().URL.Query())
			}
		})
	}
}
