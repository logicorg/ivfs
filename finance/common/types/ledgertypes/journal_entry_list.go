package ledgertypes

import (
	"fmt"
	"time"

	"cuelang.org/go/cuego"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

const (
	// DefaultJournalEntriesLimit is the page size when Limit is not set.
	DefaultJournalEntriesLimit = 100
	// MaxJournalEntriesLimit is the largest page a caller can ask for.
	MaxJournalEntriesLimit = 500
)

// JournalEntryCursor is where a page of entries ends, and the next one starts
// after it. It is the position of the last entry returned in the order of the
// list, which is by date and then by id.
type JournalEntryCursor struct {
	OccurredAt time.Time `json:"occurredAt"`
	ID         int64     `json:"id"`
}

// ListJournalEntriesParams selects a page of journal entries. A zero filter is
// not applied, and the filters that are given all have to match.
//
// The date range is half-open on the date the event occurred: From is included
// and To is not, so consecutive ranges neither overlap nor leave a gap. Entries
// are listed oldest first, or newest first when Descending is set, and entries
// of the same instant are ordered by id in the same direction.
type ListJournalEntriesParams struct {
	// LedgerAccountID selects the entries that have a posting on this account.
	LedgerAccountID int64 `json:"ledgerAccountId" cue:">=0"`
	CounterpartyID  int64 `json:"counterpartyId" cue:">=0"`
	// ExternalID has to match the external id of the entry exactly.
	ExternalID string `json:"externalId"`
	// Statuses selects the entries in any of these states, and every entry when
	// it is empty.
	Statuses   []model.JournalEntryStatus `json:"statuses,omitempty"`
	From       time.Time                  `json:"from"`
	To         time.Time                  `json:"to"`
	Descending bool                       `json:"descending"`
	// Limit is the size of the page, DefaultJournalEntriesLimit when zero.
	Limit int `json:"limit" cue:">=0"`
	// After is the Next of the previous page, and is nil for the first one.
	After *JournalEntryCursor `json:"after,omitempty"`
}

// ListJournalEntriesResult is one page of entries, each with its postings in
// line order. Next is nil on the last page.
type ListJournalEntriesResult struct {
	Entries []model.JournalEntry `json:"entries"`
	Next    *JournalEntryCursor  `json:"next,omitempty"`
}

// PageSize returns the number of entries to return in a page.
func (p *ListJournalEntriesParams) PageSize() int {
	if p.Limit == 0 {
		return DefaultJournalEntriesLimit
	}
	return p.Limit
}

// Validate checks the parameters without the database.
func (p *ListJournalEntriesParams) Validate() error {
	if err := cuego.Validate(p); err != nil {
		return internalerrors.ValidationError{Err: err}
	}
	if p.Limit > MaxJournalEntriesLimit {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("limit must not be more than %d", MaxJournalEntriesLimit)}
	}
	if !p.From.IsZero() && !p.To.IsZero() && !p.To.After(p.From) {
		return internalerrors.ErrInvalidParameter{Description: "the end of the date range must be after its start"}
	}
	for _, status := range p.Statuses {
		if !model.IsValidJournalEntryStatus(status) {
			return internalerrors.ErrInvalidParameter{Description: "invalid journal entry status"}
		}
	}
	if p.After != nil && (p.After.ID <= 0 || p.After.OccurredAt.IsZero()) {
		return internalerrors.ErrInvalidParameter{Description: "the cursor needs the date and the id of an entry"}
	}
	return nil
}
