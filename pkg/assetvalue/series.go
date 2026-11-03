package assetvalue

import (
	"fmt"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/errors"
)

// Interval is how far apart the dates of a range are.
type Interval string

// The intervals of the spec, each answered on its last day.
const (
	Daily   Interval = "day"
	Weekly  Interval = "week"
	Monthly Interval = "month"
)

// Range is the dates asked for: the last day of each interval from From to
// To, and To itself.
type Range struct {
	From, To time.Time
	// Interval is Daily when empty.
	Interval Interval
}

// normalised returns r with its bounds as days and its interval set.
func (r Range) normalised() (Range, error) {
	r.From, r.To = Day(r.From), Day(r.To)
	if r.Interval == "" {
		r.Interval = Daily
	}
	switch {
	case r.Interval != Daily && r.Interval != Weekly && r.Interval != Monthly:
		return Range{}, errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the interval %q is not one of %s, %s and %s", r.Interval, Daily, Weekly, Monthly)}
	case r.To.Before(r.From):
		return Range{}, errors.ErrInvalidParameter{Description: fmt.Sprintf(
			"the range ends on %s, before it starts on %s", r.To.Format(time.DateOnly), r.From.Format(time.DateOnly))}
	}
	return r, nil
}

// Dates returns the dates of r, as midnight UTC, oldest first.
func (r Range) Dates() ([]time.Time, error) {
	r, err := r.normalised()
	if err != nil {
		return nil, err
	}
	return r.dates(), nil
}

// dates returns the dates of r, which is normalised.
func (r Range) dates() []time.Time {
	var dates []time.Time
	for date := r.end(r.From); !date.After(r.To); date = r.end(date.AddDate(0, 0, 1)) {
		dates = append(dates, date)
	}
	// The range rarely ends on the last day of an interval, and its latest
	// value is the one most wanted.
	if len(dates) == 0 || !dates[len(dates)-1].Equal(r.To) {
		dates = append(dates, r.To)
	}
	return dates
}

// end returns the last day of the interval holding day: weeks run Monday to
// Sunday, as in ISO 8601.
func (r Range) end(day time.Time) time.Time {
	switch r.Interval {
	case Weekly:
		return day.AddDate(0, 0, (7-int(day.Weekday()))%7)
	case Monthly:
		return time.Date(day.Year(), day.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	default:
		return day
	}
}
