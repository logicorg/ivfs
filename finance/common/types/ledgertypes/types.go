package ledgertypes

import (
	"slices"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
)

// PostingInput holds the inputs to post a Posting.
type PostingInput struct {
	LedgerAccountID int64             `json:"ledgerAccountId"`
	Amount          decimal.Decimal   `json:"amount"`
	Side            model.PostingSide `json:"side"`
	// CategoryID classifies a posting to an income or expense account, and is
	// nil on any other posting and on an uncategorized one.
	CategoryID *int64 `json:"categoryId,omitempty"`
}

// PostJournalEntryInput is the input struct to post a JornalEntry.
type PostJournalEntryInput struct {
	IdempotencyKey string            `json:"idempotencyKey"`
	OccurredAt     time.Time         `json:"occurredAt"`
	Description    *string           `json:"description,omitempty"`
	CounterpartyID *int64            `json:"counterpartyId,omitempty"`
	ExternalID     *string           `json:"externalId,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
	Postings       []PostingInput    `json:"postings"`
}

// Validate performs some basic validation checks on a PostJournalEntryInput
func (pi *PostJournalEntryInput) Validate() error {
	if pi.OccurredAt.IsZero() || len(pi.Postings) < 2 {
		return internalerrors.ErrInvalidParameter{Description: "entry requires an occurrence date and at least two postings"}
	}

	for _, p := range pi.Postings {
		if p.LedgerAccountID <= 0 || !p.Amount.IsPositive() || !model.IsValidPostingSide(p.Side) {
			return internalerrors.ErrInvalidParameter{Description: "invalid posting"}
		}
		if p.CategoryID != nil && *p.CategoryID <= 0 {
			return internalerrors.ErrInvalidParameter{Description: "invalid posting category"}
		}
	}

	return nil
}

// AccountIDs returns an array holding all the accountIDs of the JornalEntry postings.
func (pi *PostJournalEntryInput) AccountIDs() []int64 {
	accountIDs := make([]int64, 0, len(pi.Postings))
	seen := make(map[int64]bool, len(pi.Postings))

	for _, p := range pi.Postings {
		if _, ok := seen[p.LedgerAccountID]; !ok {
			seen[p.LedgerAccountID] = true
			accountIDs = append(accountIDs, p.LedgerAccountID)
		}
	}

	return accountIDs
}

// CategoryIDs returns the categories the postings carry, each once.
func (pi *PostJournalEntryInput) CategoryIDs() []int64 {
	categoryIDs := make([]int64, 0, len(pi.Postings))
	for _, p := range pi.Postings {
		if p.CategoryID != nil && !slices.Contains(categoryIDs, *p.CategoryID) {
			categoryIDs = append(categoryIDs, *p.CategoryID)
		}
	}
	return categoryIDs
}

type PostJournalEntryResult struct {
	JournalEntryID int64 `json:"journalEntryId"`
}

// RetrieveJournalEntryParams holds the params to retrieve a JournalEntry.
// TODO: cuego validation
type RetrieveJournalEntryParams struct {
	ID int64 `json:"id"`
}

// RetrieveJournalEntryResult holds a retrieved JornalEntry.
type RetrieveJournalEntryResult struct {
	JournalEntry *model.JournalEntry `json:"journalEntry"`
}

// LedgerAccountBalanceParams selects the ledger account to report the
// balance of, and the instant to report it at.
type LedgerAccountBalanceParams struct {
	LedgerAccountID int64     `json:"ledgerAccountId"`
	At              time.Time `json:"at,omitzero" jsonschema:"count only entries that occurred at or before this RFC 3339 instant, such as 2026-01-31T23:59:59Z; now if omitted"`
}

// LedgerAccountBalanceResult is the balance of one ledger account in its
// natural direction. It carries the account's name, kind, and currency code
// beside their ids, so a reader can present the balance without looking them
// up, and whether the account is closed.
type LedgerAccountBalanceResult struct {
	LedgerAccountID   int64                   `json:"ledgerAccountId"`
	LedgerAccountName string                  `json:"ledgerAccountName"`
	Kind              model.LedgerAccountKind `json:"kind"`
	CurrencyID        int64                   `json:"currencyId"`
	CurrencyCode      string                  `json:"currencyCode"`
	Balance           decimal.Decimal         `json:"balance"`
	Closed            bool                    `json:"closed"`
}
