package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/shopspring/decimal"
)

// PostJournalEntryToolInput mirrors PostJournalEntryInput with
// IdempotencyKey dropped: the workflow fills a random one in when it is left
// empty (workflows.DefaultIdempotencyKey), so an MCP caller never needs to
// invent one. Field names must match PostJournalEntryInput's JSON tags,
// since Temporal deserializes this value into that type on the worker side -
// a field missing here just deserializes to its zero value there.
type PostJournalEntryToolInput struct {
	ReversalOfEntryID *int64            `json:"reversalOfEntryId,omitempty"`
	OccurredAt        time.Time         `json:"occurredAt"`
	Description       *string           `json:"description,omitempty"`
	CounterpartyID    *int64            `json:"counterpartyId,omitempty"`
	ExternalID        *string           `json:"externalId,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Postings          []lt.PostingInput `json:"postings"`
}

// ReverseJournalEntryToolInput mirrors ReverseJournalEntryInput with
// IdempotencyKey dropped, for the same reason as PostJournalEntryToolInput.
type ReverseJournalEntryToolInput struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// PostingSummary and JournalEntrySummary are flat projections of
// model.Posting and model.JournalEntry. The models themselves cannot be used
// as an MCP tool's output type directly: JournalEntry.Postings <->
// Posting.JournalEntry and JournalEntry.ReversalOfEntry form cycles, and
// jsonschema-go's reflection-based schema inference panics on a cyclic Go
// type rather than emitting a recursive schema.

type PostingSummary struct {
	LedgerAccountID int64           `json:"ledgerAccountId"`
	LineNumber      int             `json:"lineNumber"`
	Amount          decimal.Decimal `json:"amount"`
	Side            string          `json:"side"`
}

type JournalEntrySummary struct {
	ID                int64            `json:"id"`
	Status            string           `json:"status"`
	OccurredAt        time.Time        `json:"occurredAt"`
	Description       *string          `json:"description,omitempty"`
	CounterpartyID    *int64           `json:"counterpartyId,omitempty"`
	ExternalID        *string          `json:"externalId,omitempty"`
	ReversalOfEntryID *int64           `json:"reversalOfEntryId,omitempty"`
	Postings          []PostingSummary `json:"postings"`
}

// ListJournalEntriesToolResult is list_journal_entries's output.
// JournalEntryCursor has no relation fields, so it is reused as-is for
// pagination rather than mirrored.
type ListJournalEntriesToolResult struct {
	Entries []JournalEntrySummary  `json:"entries"`
	Next    *lt.JournalEntryCursor `json:"next,omitempty"`
}
