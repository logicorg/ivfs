package assetvalue

import "time"

// Dated is a row answering for a date.
type Dated[T any] struct {
	Date time.Time
	Row  T
}

// CarryForward pairs each date with the latest of rows, which are oldest
// first, observed before the date's UTC day ends: the spec's value of a
// date, so a Saturday repeats Friday's. Dates before the first row are left
// out: the series starts when the values do.
func CarryForward[T any](dates []time.Time, rows []T, observedAt func(T) time.Time) []Dated[T] {
	series := make([]Dated[T], 0, len(dates))
	next := 0
	for _, date := range dates {
		end := date.AddDate(0, 0, 1)
		for next < len(rows) && observedAt(rows[next]).Before(end) {
			next++
		}
		if next > 0 {
			series = append(series, Dated[T]{Date: date, Row: rows[next-1]})
		}
	}
	return series
}
