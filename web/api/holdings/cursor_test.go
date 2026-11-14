package holdings

import (
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/stretchr/testify/require"
)

// TestCursor checks that a cursor decodes back to itself, and that only what
// encodeCursor returns decodes.
func TestCursor(t *testing.T) {
	occurredAt := time.Date(2026, 1, 31, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 2, 3, 9, 30, 0, 123456000, time.UTC)

	tests := map[string]struct {
		after string
		want  *lt.PositionMovementCursor
	}{
		"a cursor": {
			after: "2026-01-31T10:00:00Z_2026-02-03T09:30:00.123456Z_7",
			want:  &lt.PositionMovementCursor{OccurredAt: occurredAt, CreatedAt: createdAt, ID: 7},
		},
		"nothing":                       {after: ""},
		"no creation time":              {after: "2026-01-31T10:00:00Z_7"},
		"no id":                         {after: "2026-01-31T10:00:00Z_2026-02-03T09:30:00.123456Z"},
		"an id of zero":                 {after: "2026-01-31T10:00:00Z_2026-02-03T09:30:00.123456Z_0"},
		"a date that is not an instant": {after: "2026-01-31_2026-02-03T09:30:00.123456Z_7"},
		"a creation time that is not":   {after: "2026-01-31T10:00:00Z_yesterday_7"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, ok := decodeCursor(tc.after)
			require.Equal(t, tc.want != nil, ok)
			require.Equal(t, tc.want, got)
			if ok {
				require.Equal(t, tc.after, encodeCursor(*got))
			}
		})
	}
}
