package journal

import (
	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// encodeCursor is cursor as the after of the next page, or empty on the last.
func encodeCursor(cursor *lt.JournalEntryCursor) string {
	if cursor == nil {
		return ""
	}
	return gateway.EncodeCursor(cursor.OccurredAt, cursor.ID)
}

// decodeCursor is the cursor after names, nil for the first page, or false if
// after is not one encodeCursor returns.
func decodeCursor(after string) (*lt.JournalEntryCursor, bool) {
	if after == "" {
		return nil, true
	}
	occurredAt, id, ok := gateway.DecodeCursor(after)
	if !ok {
		return nil, false
	}
	return &lt.JournalEntryCursor{OccurredAt: occurredAt, ID: id}, true
}
