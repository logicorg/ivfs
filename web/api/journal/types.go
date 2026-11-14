package journal

import (
	"fmt"
	"time"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/web/api/gateway"
)

// Posting is one line of a journal entry: an amount debited or credited to a
// ledger account.
type Posting struct {
	LineNumber        int    `json:"lineNumber"`
	LedgerAccountID   int64  `json:"ledgerAccountId"`
	LedgerAccountName string `json:"ledgerAccountName"`
	Kind              string `json:"kind" enum:"asset,liability,equity,income,expense"`
	// CurrencyID is the ledger account's currency, which the amount is in.
	CurrencyID int64          `json:"currencyId"`
	Amount     gateway.Amount `json:"amount"`
	Side       string         `json:"side" enum:"debit,credit"`
	// CategoryID and CategoryName are absent on a posting without a category:
	// one to an account that is neither income nor expense, or an
	// uncategorized one.
	CategoryID   *int64 `json:"categoryId,omitempty"`
	CategoryName string `json:"categoryName,omitempty"`
}

// Entry is a journal entry with its postings, in line order.
type Entry struct {
	ID          int64     `json:"id"`
	OccurredAt  time.Time `json:"occurredAt"`
	Description string    `json:"description,omitempty"`
	ExternalID  string    `json:"externalId,omitempty"`
	Status      string    `json:"status" enum:"draft,posted"`
	Postings    []Posting `json:"postings"`
}

// EntryPage is one page of journal entries.
type EntryPage struct {
	Entries []Entry `json:"entries"`
	// Next is the after of the next page, and is absent on the last one.
	Next string `json:"next,omitempty"`
}

// EntryDetail is a journal entry with what only its detail shows.
type EntryDetail struct {
	Entry
	CounterpartyID   *int64     `json:"counterpartyId,omitempty"`
	CounterpartyName string     `json:"counterpartyName,omitempty"`
	PostedAt         *time.Time `json:"postedAt,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
}

// ListJournalEntriesInput is the request of GET /api/journal-entries.
type ListJournalEntriesInput struct {
	LedgerAccountID int64     `query:"ledgerAccountId" minimum:"1" doc:"only entries with a posting on this ledger account"`
	CounterpartyID  int64     `query:"counterpartyId" minimum:"1" doc:"only entries with this counterparty"`
	Statuses        []string  `query:"status,explode" enum:"draft,posted" doc:"only entries in any of these states; every state if omitted"`
	From            time.Time `query:"from" doc:"only entries that occurred at or after this instant"`
	To              time.Time `query:"to" doc:"only entries that occurred before this instant"`
	Descending      bool      `query:"descending" doc:"newest first"`
	Limit           int       `query:"limit" minimum:"1" maximum:"500" doc:"the size of the page; 100 if omitted"`
	After           string    `query:"after" doc:"the next of the previous page; the first page if omitted"`
}

// ListJournalEntriesOutput is the response of GET /api/journal-entries.
type ListJournalEntriesOutput struct {
	Body EntryPage
}

// RetrieveJournalEntryInput is the request of GET /api/journal-entries/{id}.
type RetrieveJournalEntryInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// RetrieveJournalEntryOutput is the response of GET
// /api/journal-entries/{id}.
type RetrieveJournalEntryOutput struct {
	Body EntryDetail
}

// DeleteJournalEntryInput is the request of DELETE /api/journal-entries/{id}.
type DeleteJournalEntryInput struct {
	ID int64 `path:"id" minimum:"1"`
}

// DeleteJournalEntryOutput is the response of DELETE
// /api/journal-entries/{id}.
type DeleteJournalEntryOutput struct {
	Body gateway.Deletion
}

// newEntry is entry as the API returns it. Every workflow that
// returns entries loads each posting's ledger account, so one missing is a
// bug.
func newEntry(entry *model.JournalEntry) (Entry, error) {
	out := Entry{
		ID: entry.ID, OccurredAt: entry.OccurredAt, Status: string(entry.Status),
		Postings: make([]Posting, 0, len(entry.Postings)),
	}
	if entry.Description != nil {
		out.Description = *entry.Description
	}
	if entry.ExternalID != nil {
		out.ExternalID = *entry.ExternalID
	}
	for _, posting := range entry.Postings {
		if posting.LedgerAccount == nil {
			return Entry{}, fmt.Errorf("posting %d of journal entry %d has no ledger account loaded", posting.ID, entry.ID)
		}
		line := Posting{
			LineNumber: posting.LineNumber, LedgerAccountID: posting.LedgerAccountID,
			LedgerAccountName: posting.LedgerAccount.Name, Kind: string(posting.LedgerAccount.Kind),
			CurrencyID: posting.LedgerAccount.CurrencyID, Amount: gateway.NewAmount(posting.Amount), Side: string(posting.Side),
		}
		if posting.CategoryID != nil {
			if posting.Category == nil {
				return Entry{}, fmt.Errorf("posting %d of journal entry %d has no category loaded", posting.ID, entry.ID)
			}
			line.CategoryID, line.CategoryName = posting.CategoryID, posting.Category.Name
		}
		out.Postings = append(out.Postings, line)
	}
	return out, nil
}
