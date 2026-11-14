package accounts

import (
	"context"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/pkg/model"
)

// ClearCounterpartyReferences removes the counterparty from every journal entry
// that refers to it, so that it can be deleted (ADR 0016). Postings and
// movements are untouched, so no balance or holding changes. It is one
// statement and needs no transaction. A retry finds nothing left to clear and
// reports 0.
func (a *Activities) ClearCounterpartyReferences(ctx context.Context, input lt.ClearCounterpartyReferencesInput) (*lt.ClearCounterpartyReferencesResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	res, err := db.NewUpdate().Model((*model.JournalEntry)(nil)).
		Set("counterparty_id = NULL").
		Set("updated_at = now()").
		Where("counterparty_id = ?", input.CounterpartyID).
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("clear counterparty references: %w", err)
	}
	cleared, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("count cleared counterparty references: %w", err)
	}
	return &lt.ClearCounterpartyReferencesResult{Cleared: cleared}, nil
}
