package assetvalue

import (
	"testing"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestRangeDates(t *testing.T) {
	tests := map[string]struct {
		r       Range
		want    []time.Time
		wantErr string
	}{
		"every day of a weekend": {
			r:    Range{From: date(2026, 10, 9), To: date(2026, 10, 11), Interval: Daily},
			want: []time.Time{date(2026, 10, 9), date(2026, 10, 10), date(2026, 10, 11)},
		},
		"every day, when the interval is not set": {
			r:    Range{From: date(2026, 10, 9), To: date(2026, 10, 10)},
			want: []time.Time{date(2026, 10, 9), date(2026, 10, 10)},
		},
		"a single day": {
			r:    Range{From: date(2026, 10, 10), To: date(2026, 10, 10), Interval: Monthly},
			want: []time.Time{date(2026, 10, 10)},
		},
		"every Sunday, then the Wednesday the range ends on": {
			r:    Range{From: date(2026, 10, 7), To: date(2026, 10, 21), Interval: Weekly},
			want: []time.Time{date(2026, 10, 11), date(2026, 10, 18), date(2026, 10, 21)},
		},
		"every Sunday, the last one ending the range": {
			r:    Range{From: date(2026, 10, 7), To: date(2026, 10, 18), Interval: Weekly},
			want: []time.Time{date(2026, 10, 11), date(2026, 10, 18)},
		},
		"a week that starts on a Sunday": {
			r:    Range{From: date(2026, 10, 11), To: date(2026, 10, 12), Interval: Weekly},
			want: []time.Time{date(2026, 10, 11), date(2026, 10, 12)},
		},
		"within a week": {
			r:    Range{From: date(2026, 10, 12), To: date(2026, 10, 14), Interval: Weekly},
			want: []time.Time{date(2026, 10, 14)},
		},
		"the end of every month, across a year": {
			r:    Range{From: date(2025, 11, 15), To: date(2026, 2, 10), Interval: Monthly},
			want: []time.Time{date(2025, 11, 30), date(2025, 12, 31), date(2026, 1, 31), date(2026, 2, 10)},
		},
		"the end of February in a leap year": {
			r:    Range{From: date(2028, 2, 1), To: date(2028, 3, 31), Interval: Monthly},
			want: []time.Time{date(2028, 2, 29), date(2028, 3, 31)},
		},
		"bounds in another zone, by their own date": {
			r: Range{
				From: time.Date(2026, 10, 9, 1, 0, 0, 0, time.FixedZone("CEST", 2*60*60)),
				To:   time.Date(2026, 10, 9, 23, 0, 0, 0, time.FixedZone("EDT", -4*60*60)),
			},
			want: []time.Time{date(2026, 10, 9)},
		},
		"an interval the spec does not have": {
			r:       Range{From: date(2026, 10, 9), To: date(2026, 10, 11), Interval: "year"},
			wantErr: `the interval "year" is not one of day, week and month`,
		},
		"a range that ends before it starts": {
			r:       Range{From: date(2026, 10, 11), To: date(2026, 10, 9)},
			wantErr: "the range ends on 2026-10-09, before it starts on 2026-10-11",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tc.r.Dates()

			if tc.wantErr != "" {
				var invalid errors.ErrInvalidParameter
				require.ErrorAs(t, err, &invalid)
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}
