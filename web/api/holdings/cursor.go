package holdings

import (
	"strings"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// encodeCursor is cursor as the after of the next page. The movements are
// ordered by date, then by when they were recorded, then by id (ADR 0017), so
// it is the date followed by a gateway cursor of the other two, such as
// "2026-01-31T10:00:00Z_2026-02-03T09:30:00Z_42".
func encodeCursor(cursor lt.PositionMovementCursor) string {
	return cursor.OccurredAt.UTC().Format(time.RFC3339Nano) + "_" + gateway.EncodeCursor(cursor.CreatedAt, cursor.ID)
}

// decodeCursor is the cursor after names, or false if after is not one
// encodeCursor returns.
func decodeCursor(after string) (*lt.PositionMovementCursor, bool) {
	instant, rest, found := strings.Cut(after, "_")
	if !found {
		return nil, false
	}
	occurredAt, err := time.Parse(time.RFC3339Nano, instant)
	if err != nil {
		return nil, false
	}
	createdAt, id, ok := gateway.DecodeCursor(rest)
	if !ok {
		return nil, false
	}
	return &lt.PositionMovementCursor{OccurredAt: occurredAt, CreatedAt: createdAt, ID: id}, true
}
