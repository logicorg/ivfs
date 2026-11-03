package gateway

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestCursor checks that a cursor decodes back to itself, and that only what
// EncodeCursor returns decodes.
func TestCursor(t *testing.T) {
	tests := map[string]struct {
		after          string
		wantOccurredAt time.Time
		wantID         int64
		wantOK         bool
	}{
		"a cursor": {
			after: "2026-01-31T10:00:00Z_7", wantOccurredAt: time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC), wantID: 7, wantOK: true,
		},
		"a cursor to the nanosecond": {
			after: "2026-01-31T10:00:00.000000001Z_7", wantOccurredAt: time.Date(2026, 1, 31, 10, 0, 0, 1, time.UTC), wantID: 7,
			wantOK: true,
		},
		"nothing":           {after: ""},
		"no id":             {after: "2026-01-31T10:00:00Z"},
		"an id of zero":     {after: "2026-01-31T10:00:00Z_0"},
		"an id that is not": {after: "2026-01-31T10:00:00Z_seven"},
		"no instant":        {after: "_7"},
		"a date only":       {after: "2026-01-31_7"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			occurredAt, id, ok := DecodeCursor(tc.after)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.wantOccurredAt, occurredAt)
			require.Equal(t, tc.wantID, id)
			if ok {
				require.Equal(t, tc.after, EncodeCursor(occurredAt, id))
			}
		})
	}
}
