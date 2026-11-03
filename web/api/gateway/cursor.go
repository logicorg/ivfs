package gateway

import (
	"strconv"
	"strings"
	"time"
)

// EncodeCursor is the after of the next page of a history ordered by date and
// then by id: the instant and the id of the last item of this one, such as
// "2026-01-31T10:00:00Z_42". The UI passes it back as it is.
func EncodeCursor(occurredAt time.Time, id int64) string {
	return occurredAt.UTC().Format(time.RFC3339Nano) + "_" + strconv.FormatInt(id, 10)
}

// DecodeCursor is the instant and the id after names, or false if after is
// not one EncodeCursor returns. An empty after is the first page, which has no
// cursor, so the caller handles it before.
func DecodeCursor(after string) (time.Time, int64, bool) {
	instant, id, found := strings.Cut(after, "_")
	if !found {
		return time.Time{}, 0, false
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, instant)
	if err != nil {
		return time.Time{}, 0, false
	}
	itemID, err := strconv.ParseInt(id, 10, 64)
	if err != nil || itemID <= 0 {
		return time.Time{}, 0, false
	}
	return occurredAt, itemID, true
}
