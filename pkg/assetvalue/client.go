package assetvalue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"time"
)

// maxBodySize bounds what is read of a provider's answer: about 120 bytes a
// date, so a century of days.
const maxBodySize = 4 << 20

// Client is a Provider reached over HTTP, through v1 of the spec.
type Client struct {
	base       *url.URL
	httpClient *http.Client
}

// NewClient returns the client of the provider at baseURL. httpClient
// should have no timeout of its own: a request is bounded by its context.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(baseURL)
	switch {
	case err != nil:
		return nil, err
	case (u.Scheme != "http" && u.Scheme != "https") || u.Host == "":
		return nil, fmt.Errorf("%q is not an absolute URL, such as http://yahoo-provider:8080", baseURL)
	case u.RawQuery != "" || u.Fragment != "" || u.User != nil:
		return nil, fmt.Errorf("%q must be a base URL alone: the client adds /v1/values and /v1/rates to it", baseURL)
	}
	return &Client{base: u, httpClient: httpClient}, nil
}

type problem struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// get asks the provider for the endpoint at path, with query and r, and
// returns its answer when it has one, or notFound when it answers 404.
func (c *Client) get(ctx context.Context, path string, query url.Values, r Range, notFound error) ([]byte, error) {
	query.Set("from", r.From.Format(time.DateOnly))
	query.Set("to", r.To.Format(time.DateOnly))
	query.Set("interval", string(r.Interval))
	endpoint := c.base.JoinPath("v1", path)
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("build the request: %w", err)
	}
	req.Header.Set("Accept", "application/json, application/problem+json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call the provider: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize+1))
	if err != nil {
		return nil, fmt.Errorf("read the provider's answer: %w", err)
	}
	if len(body) > maxBodySize {
		return nil, fmt.Errorf("the provider's answer is larger than %d MiB; ask for a shorter range or a longer interval",
			maxBodySize>>20)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		return body, nil
	case http.StatusNotFound:
		return nil, notFound
	default:
		return nil, fmt.Errorf("the provider answered %s%s", resp.Status, problemDetail(resp, body))
	}
}

// checkDates returns the dates the provider answered, refusing them unless
// they are the dates of r, want, from the first one it has a value for.
func checkDates(answered []string, r Range, want []time.Time) ([]time.Time, error) {
	if len(answered) == 0 {
		return nil, fmt.Errorf("the provider answered nothing, which breaks the spec: it answers 404 instead")
	}
	dates := make([]time.Time, len(answered))
	first := -1
	for i, a := range answered {
		date, err := time.Parse(time.DateOnly, a)
		if err != nil {
			return nil, fmt.Errorf("the provider answered the date %q, which breaks the spec: "+
				"it must be a date, such as 2026-10-10", a)
		}
		if date.Before(r.From) || date.After(r.To) {
			return nil, fmt.Errorf("the provider answered %s, outside the range from %s to %s",
				a, r.From.Format(time.DateOnly), r.To.Format(time.DateOnly))
		}
		if i == 0 {
			for j, w := range want {
				if w.Equal(date) {
					first = j
				}
			}
			if first < 0 {
				return nil, fmt.Errorf("the provider answered %s, which breaks the spec: "+
					"it is neither the last day of a %s nor the end of the range", a, r.Interval)
			}
		} else if first+i >= len(want) || !date.Equal(want[first+i]) {
			return nil, fmt.Errorf("the provider answered %s after %s, which breaks the spec: "+
				"it answers one value a %s, oldest first, with none missing", a, answered[i-1], r.Interval)
		}
		dates[i] = date
	}
	if first+len(answered) < len(want) {
		return nil, fmt.Errorf("the provider stops on %s, which breaks the spec: it goes on to %s",
			answered[len(answered)-1], r.To.Format(time.DateOnly))
	}
	return dates, nil
}

// checkObserved refuses observedAt as the time of the value on date unless
// it is before the day ends.
func checkObserved(date, observedAt time.Time) error {
	if observedAt.IsZero() {
		return fmt.Errorf("the provider answered no observedAt for %s, which the spec requires", date.Format(time.DateOnly))
	}
	if !observedAt.Before(date.AddDate(0, 0, 1)) {
		return fmt.Errorf("the provider answered a value for %s observed at %s, after the day ended",
			date.Format(time.DateOnly), observedAt.Format(time.RFC3339))
	}
	return nil
}

// problemDetail returns what the problem in body says, as ": detail", or
// nothing when body is not a problem.
func problemDetail(resp *http.Response, body []byte) string {
	mediaType, _, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if err != nil || (mediaType != "application/problem+json" && mediaType != "application/json") {
		return ""
	}
	var p problem
	if json.Unmarshal(body, &p) != nil {
		return ""
	}
	if p.Detail != "" {
		return ": " + p.Detail
	}
	if p.Title != "" {
		return ": " + p.Title
	}
	return ""
}
