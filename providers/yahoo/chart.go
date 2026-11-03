package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"sync"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/shopspring/decimal"
)

// cacheTTL is how long an answer of Yahoo's is reused: the holdings screen
// asks for today's value of every asset on each visit, and Yahoo rate
// limits by address.
const cacheTTL = 10 * time.Minute

// maxChartSize bounds what is read of Yahoo's answer: about 100 bytes a day,
// so a century of days and more.
const maxChartSize = 8 << 20

// userAgent is sent because Yahoo answers 429 to a request without one.
const userAgent = "Mozilla/5.0 (compatible; ivfs-yahoo-provider/1)"

// bar is a symbol's close of one session, or its latest trade while the
// session is open.
type bar struct {
	close      decimal.Decimal
	observedAt time.Time
}

// chart is a symbol's bars, oldest first.
type chart struct {
	// currency is as Yahoo writes it, which may be a minor unit, such as GBp.
	currency string
	// places is how many decimal places Yahoo shows the symbol's price with.
	places int32
	bars   []bar
}

// chartClient reads daily bars from Yahoo's chart API, the one its own pages
// use. It is unofficial and unauthenticated, so it may change or rate limit
// without notice.
type chartClient struct {
	baseURL    *url.URL
	httpClient *http.Client
	now        func() time.Time

	mu    sync.Mutex
	cache map[chartKey]cachedChart
}

type chartKey struct {
	symbol   string
	from, to time.Time
}

// cachedChart is an answer, a symbol Yahoo does not know included.
type cachedChart struct {
	chart   chart
	err     error
	expires time.Time
}

func newChartClient(baseURL *url.URL, httpClient *http.Client) *chartClient {
	return &chartClient{baseURL: baseURL, httpClient: httpClient, now: time.Now, cache: make(map[chartKey]cachedChart)}
}

// wireChart is the part of Yahoo's answer read. A close stays the number
// Yahoo wrote until rounded, and is null for a session with no trades.
type wireChart struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency             string `json:"currency"`
				PriceHint            *int32 `json:"priceHint"`
				RegularMarketTime    int64  `json:"regularMarketTime"`
				CurrentTradingPeriod struct {
					Regular struct {
						Start int64 `json:"start"`
						End   int64 `json:"end"`
					} `json:"regular"`
				} `json:"currentTradingPeriod"`
			} `json:"meta"`
			Timestamp  []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []*json.Number `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// closes returns symbol's daily bars of the sessions from from to to, both
// midnight UTC and both included, or ierrors.ErrNotFound when Yahoo does not
// know the symbol.
func (c *chartClient) closes(ctx context.Context, symbol string, from, to time.Time) (chart, error) {
	key := chartKey{symbol: symbol, from: from, to: to}
	c.mu.Lock()
	hit, ok := c.cache[key]
	c.mu.Unlock()
	if ok && c.now().Before(hit.expires) {
		return hit.chart, hit.err
	}

	// Not cancelled with the request, so that an answer slower than the
	// gateway's timeout is still cached for its next request. The HTTP
	// client's own timeout bounds it.
	ch, err := c.fetch(context.WithoutCancel(ctx), symbol, from, to)
	var notFound ierrors.ErrNotFound
	if err != nil && !errors.As(err, &notFound) {
		return chart{}, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.now()
	for k, v := range c.cache {
		if !now.Before(v.expires) {
			delete(c.cache, k)
		}
	}
	c.cache[key] = cachedChart{chart: ch, err: err, expires: now.Add(cacheTTL)}
	return ch, err
}

// fetch asks Yahoo for symbol's daily bars from from to to.
func (c *chartClient) fetch(ctx context.Context, symbol string, from, to time.Time) (chart, error) {
	endpoint := c.baseURL.JoinPath("v8", "finance", "chart", symbol)
	endpoint.RawQuery = url.Values{
		"period1":  {strconv.FormatInt(from.Unix(), 10)},
		"period2":  {strconv.FormatInt(to.AddDate(0, 0, 1).Unix(), 10)},
		"interval": {"1d"},
	}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return chart{}, fmt.Errorf("build the request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return chart{}, fmt.Errorf("call the Yahoo chart API: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxChartSize+1))
	if err != nil {
		return chart{}, fmt.Errorf("read the Yahoo chart API's answer: %w", err)
	}
	if len(body) > maxChartSize {
		return chart{}, fmt.Errorf("the Yahoo chart API's answer for %s is larger than %d MiB; ask for a shorter range", symbol, maxChartSize>>20)
	}

	notFound := ierrors.ErrNotFound{ResourceType: "Symbol", ID: symbol}
	var wire wireChart
	decodeErr := json.Unmarshal(body, &wire)
	switch {
	case resp.StatusCode == http.StatusNotFound:
		return chart{}, notFound
	case resp.StatusCode == http.StatusTooManyRequests:
		return chart{}, fmt.Errorf("the Yahoo chart API is rate limiting this address; try again in a few minutes")
	case resp.StatusCode != http.StatusOK:
		if decodeErr == nil && wire.Chart.Error != nil {
			return chart{}, fmt.Errorf("the Yahoo chart API answered %s for %s: %s", resp.Status, symbol, wire.Chart.Error.Description)
		}
		return chart{}, fmt.Errorf("the Yahoo chart API answered %s for %s", resp.Status, symbol)
	case decodeErr != nil:
		return chart{}, fmt.Errorf("the Yahoo chart API's answer for %s is not a chart: %w", symbol, decodeErr)
	case wire.Chart.Error != nil && wire.Chart.Error.Code == "Not Found":
		return chart{}, notFound
	case wire.Chart.Error != nil:
		return chart{}, fmt.Errorf("the Yahoo chart API answered %s for %s: %s", wire.Chart.Error.Code, symbol, wire.Chart.Error.Description)
	case len(wire.Chart.Result) == 0:
		return chart{}, notFound
	}

	result := wire.Chart.Result[0]
	var closes []*json.Number
	if len(result.Indicators.Quote) > 0 {
		closes = result.Indicators.Quote[0].Close
	}
	if len(closes) != len(result.Timestamp) {
		return chart{}, fmt.Errorf("the Yahoo chart API answered %d sessions and %d closes for %s", len(result.Timestamp), len(closes), symbol)
	}
	// A daily bar is dated when its session opens, and its close is
	// observed when the session ends, which today's regular one tells the
	// length of.
	period := result.Meta.CurrentTradingPeriod.Regular
	session := max(time.Duration(period.End-period.Start)*time.Second, 0)
	lastTrade := time.Unix(result.Meta.RegularMarketTime, 0)

	ch := chart{currency: result.Meta.Currency, places: 4}
	if result.Meta.PriceHint != nil {
		ch.places = *result.Meta.PriceHint
	}
	for i, opened := range result.Timestamp {
		if closes[i] == nil {
			continue
		}
		price, err := decimal.NewFromString(closes[i].String())
		if err != nil {
			return chart{}, fmt.Errorf("parse the close %q of %s: %w", closes[i].String(), symbol, err)
		}
		observedAt := time.Unix(opened, 0).Add(session)
		// The bar of a session still open holds its latest trade, not a
		// close yet.
		if result.Meta.RegularMarketTime != 0 && observedAt.After(lastTrade) {
			observedAt = lastTrade
		}
		ch.bars = append(ch.bars, bar{close: price, observedAt: observedAt.UTC()})
	}
	slices.SortStableFunc(ch.bars, func(a, b bar) int { return a.observedAt.Compare(b.observedAt) })
	return ch, nil
}
