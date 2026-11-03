// Package dates provides utility functions when dealing with dates
package dates

import "time"

// ZeroedTime returns a time.Time with the time components (hour, minute, second, nanosecond) set to zero.
func ZeroedTime(t time.Time, loc ...*time.Location) time.Time {
	location := t.Location()
	if len(loc) > 0 && loc[0] != nil {
		location = loc[0]
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, location)
}

// DaysBetween returns the number of days between two dates.
func DaysBetween(date1 time.Time, date2 time.Time) int {
	d1 := ZeroedTime(date1, time.UTC)
	d2 := ZeroedTime(date2, time.UTC)

	return int(d1.Sub(d2) / (time.Hour * 24))
}
