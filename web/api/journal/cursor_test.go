package journal

import (
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/stretchr/testify/require"
)

// TestCursor checks that a cursor decodes back to itself, and that only what
// encodeCursor returns decodes.
func TestCursor(t *testing.T) {
	tests := map[string]struct {
		after  string
		want   *lt.JournalEntryCursor
		wantOK bool
	}{
		"the first page": {after: "", wantOK: true},
		"a cursor": {
			after:  "2026-01-31T10:00:00Z_7",
			want:   &lt.JournalEntryCursor{OccurredAt: time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC), ID: 7},
			wantOK: true,
		},
		"a cursor to the nanosecond": {
			after:  "2026-01-31T10:00:00.000000001Z_7",
			want:   &lt.JournalEntryCursor{OccurredAt: time.Date(2026, 1, 31, 10, 0, 0, 1, time.UTC), ID: 7},
			wantOK: true,
		},
		"no id":             {after: "2026-01-31T10:00:00Z"},
		"an id of zero":     {after: "2026-01-31T10:00:00Z_0"},
		"an id that is not": {after: "2026-01-31T10:00:00Z_seven"},
		"no instant":        {after: "_7"},
		"a date only":       {after: "2026-01-31_7"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := decodeCursor(tc.after)
			require.Equal(t, tc.wantOK, ok)
			require.Equal(t, tc.want, got)
			if ok {
				require.Equal(t, tc.after, encodeCursor(got))
			}
		})
	}
}
