package assetvalue

import (
	"context"
	stderrors "errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/stretchr/testify/require"
)

// provider starts a provider answering status, contentType, and body, and
// returns its client, under a prefix, and the request it received, if any.
// The request is checked after the call: a failed require in the server's
// goroutine would only show as the client reading EOF.
func provider(t *testing.T, status int, contentType, body string) (*Client, func() *http.Request) {
	t.Helper()
	var received *http.Request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(server.URL+"/prefix/", server.Client())
	require.NoError(t, err)
	return client, func() *http.Request { return received }
}

func TestClientBoundedByContext(t *testing.T) {
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	defer server.Close()
	defer close(release)
	client, err := NewClient(server.URL, server.Client())
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	day := date(2026, 10, 9)
	_, err = client.Values(ctx, model.Asset{Ticker: new("AAPL")}, Range{From: day, To: day})

	require.True(t, stderrors.Is(err, context.DeadlineExceeded), "got %v", err)
}

func TestClientAnswerTooLarge(t *testing.T) {
	client, _ := provider(t, http.StatusOK, "", strings.Repeat(" ", maxBodySize+1))

	day := date(2026, 10, 9)
	_, err := client.Rates(context.Background(), "EUR", "USD", Range{From: day, To: day})

	require.ErrorContains(t, err, "the provider's answer is larger than 4 MiB; ask for a shorter range or a longer interval")
}

func TestNewClient(t *testing.T) {
	tests := map[string]struct {
		baseURL  string
		wantBase string
		wantErr  string
	}{
		"a host":                  {baseURL: "http://yahoo-provider:8080", wantBase: "http://yahoo-provider:8080"},
		"a path, with a slash":    {baseURL: "https://prices.example/ivfs/", wantBase: "https://prices.example/ivfs/"},
		"a relative URL":          {baseURL: "yahoo-provider:8080", wantErr: "is not an absolute URL"},
		"another scheme":          {baseURL: "ftp://prices.example", wantErr: "is not an absolute URL"},
		"a query":                 {baseURL: "http://prices.example?key=1", wantErr: "must be a base URL alone"},
		"a user in the URL":       {baseURL: "http://user@prices.example", wantErr: "must be a base URL alone"},
		"a URL that cannot parse": {baseURL: "http://prices.example/%zz", wantErr: "invalid URL escape"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			client, err := NewClient(tc.baseURL, http.DefaultClient)

			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantBase, client.base.String())
		})
	}
}

func TestDay(t *testing.T) {
	tests := map[string]struct {
		t    time.Time
		want time.Time
	}{
		"midnight UTC":            {t: date(2026, 10, 9), want: date(2026, 10, 9)},
		"late in the day":         {t: time.Date(2026, 10, 9, 23, 59, 59, 0, time.UTC), want: date(2026, 10, 9)},
		"its own date, not UTC's": {t: time.Date(2026, 10, 10, 1, 0, 0, 0, time.FixedZone("CEST", 2*60*60)), want: date(2026, 10, 10)},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, Day(tc.t))
		})
	}
}
