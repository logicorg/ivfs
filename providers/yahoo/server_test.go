package main

import (
	"net/url"
	"testing"
	"time"

	ierrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/assetvalue"
	"github.com/stretchr/testify/require"
)

func TestParseRange(t *testing.T) {
	from, to := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 10, 0, 0, 0, 0, time.UTC)
	tests := map[string]struct {
		query   string
		want    assetvalue.Range
		wantErr error
	}{
		"every day, when no interval is asked for": {
			query: "from=2026-10-01&to=2026-10-10",
			want:  assetvalue.Range{From: from, To: to},
		},
		"every month": {
			query: "from=2026-10-01&to=2026-10-10&interval=month",
			want:  assetvalue.Range{From: from, To: to, Interval: assetvalue.Monthly},
		},
		"no bounds": {
			query:   "interval=day",
			wantErr: ierrors.ErrMissingRequiredParameter{Parameters: []string{"from", "to"}},
		},
		"bounds that are not dates, from reported": {
			query:   "from=yesterday&to=today",
			wantErr: ierrors.ErrInvalidParameter{Description: `from is "yesterday", not a date, such as 2026-10-10`},
		},
		"a time, not a date": {
			query:   "from=2026-10-01&to=2026-10-10T12:00:00Z",
			wantErr: ierrors.ErrInvalidParameter{Description: `to is "2026-10-10T12:00:00Z", not a date, such as 2026-10-10`},
		},
		"a range that ends before it starts": {
			query:   "from=2026-10-10&to=2026-10-01",
			wantErr: ierrors.ErrInvalidParameter{Description: "the range ends on 2026-10-01, before it starts on 2026-10-10"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			query, err := url.ParseQuery(tt.query)
			require.NoError(t, err)
			got, err := parseRange(query)
			if tt.wantErr != nil {
				require.Equal(t, tt.wantErr, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
