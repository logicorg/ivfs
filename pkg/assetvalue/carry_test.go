package assetvalue

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCarryForward(t *testing.T) {
	friday := date(2026, 10, 9)
	closing := friday.Add(20 * time.Hour)
	tests := map[string]struct {
		dates []time.Time
		rows  []time.Time
		want  []Dated[time.Time]
	}{
		"a weekend repeats Friday's close": {
			dates: []time.Time{friday, date(2026, 10, 10), date(2026, 10, 11)},
			rows:  []time.Time{closing},
			want:  []Dated[time.Time]{{friday, closing}, {date(2026, 10, 10), closing}, {date(2026, 10, 11), closing}},
		},
		"the latest of a day, not the first": {
			dates: []time.Time{friday},
			rows:  []time.Time{friday.Add(9 * time.Hour), closing},
			want:  []Dated[time.Time]{{friday, closing}},
		},
		"the last instant of a day counts for it": {
			dates: []time.Time{friday},
			rows:  []time.Time{date(2026, 10, 10).Add(-time.Nanosecond)},
			want:  []Dated[time.Time]{{friday, date(2026, 10, 10).Add(-time.Nanosecond)}},
		},
		"midnight is the next day": {
			dates: []time.Time{friday, date(2026, 10, 10)},
			rows:  []time.Time{date(2026, 10, 10)},
			want:  []Dated[time.Time]{{date(2026, 10, 10), date(2026, 10, 10)}},
		},
		"rows from before the dates": {
			dates: []time.Time{friday},
			rows:  []time.Time{date(2026, 9, 1), date(2026, 10, 1)},
			want:  []Dated[time.Time]{{friday, date(2026, 10, 1)}},
		},
		"no rows": {
			dates: []time.Time{friday},
			want:  []Dated[time.Time]{},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := CarryForward(tt.dates, tt.rows, func(row time.Time) time.Time { return row })
			require.Equal(t, tt.want, got)
		})
	}
}
