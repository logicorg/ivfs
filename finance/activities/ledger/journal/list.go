package journal

import (
	"context"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/uptrace/bun"
)

// ListJournalEntries returns one page of journal entries, each with its
// postings in line order. Entries of the same instant are ordered by id, in the
// same direction as the dates, so a page boundary can fall between them without
// skipping or repeating any.
func (a *Activities) ListJournalEntries(ctx context.Context, params lt.ListJournalEntriesParams) (*lt.ListJournalEntriesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := params.Validate(); err != nil {
		return nil, err
	}

	var entries []model.JournalEntry
	query := db.NewSelect().Model(&entries)
	if params.LedgerAccountID > 0 {
		query = query.Where("EXISTS (SELECT 1 FROM ledger.posting AS p WHERE p.journal_entry_id = je.id AND p.ledger_account_id = ?)", params.LedgerAccountID)
	}
	if params.CounterpartyID > 0 {
		query = query.Where("je.counterparty_id = ?", params.CounterpartyID)
	}
	if params.ExternalID != "" {
		query = query.Where("je.external_id = ?", params.ExternalID)
	}
	if len(params.Statuses) > 0 {
		query = query.Where("je.status IN (?)", bun.List(params.Statuses))
	}
	if !params.From.IsZero() {
		query = query.Where("je.occurred_at >= ?", params.From)
	}
	if !params.To.IsZero() {
		query = query.Where("je.occurred_at < ?", params.To)
	}

	after, order := "(je.occurred_at, je.id) > (?, ?)", "je.occurred_at ASC, je.id ASC"
	if params.Descending {
		after, order = "(je.occurred_at, je.id) < (?, ?)", "je.occurred_at DESC, je.id DESC"
	}
	if params.After != nil {
		query = query.Where(after, params.After.OccurredAt, params.After.ID)
	}

	// One more than the page is read to know whether there is a next page.
	pageSize := params.PageSize()
	if err := query.OrderExpr(order).Limit(int64(pageSize) + 1).Scan(ctx); err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}

	result := &lt.ListJournalEntriesResult{Entries: entries}
	if len(entries) > pageSize {
		result.Entries = entries[:pageSize]
		last := result.Entries[pageSize-1]
		result.Next = &lt.JournalEntryCursor{OccurredAt: last.OccurredAt, ID: last.ID}
	}

	if err := loadPostings(ctx, db, result.Entries); err != nil {
		return nil, err
	}
	return result, nil
}

// loadPostings sets the postings of every entry with one query.
func loadPostings(ctx context.Context, db bun.IDB, entries []model.JournalEntry) error {
	if len(entries) == 0 {
		return nil
	}

	ids := make([]int64, len(entries))
	for n := range entries {
		ids[n] = entries[n].ID
	}
	var postings []model.Posting
	err := db.NewSelect().Model(&postings).
		Relation("LedgerAccount").
		Where("p.journal_entry_id IN (?)", bun.List(ids)).
		OrderExpr("p.journal_entry_id, p.line_number").
		Scan(ctx)
	if err != nil {
		return fmt.Errorf("load postings of journal entries: %w", err)
	}

	byEntry := make(map[int64][]model.Posting, len(entries))
	for _, posting := range postings {
		byEntry[posting.JournalEntryID] = append(byEntry[posting.JournalEntryID], posting)
	}
	for n := range entries {
		entries[n].Postings = byEntry[entries[n].ID]
	}
	return nil
}
