package journal

import (
	"context"
	"database/sql"
	"fmt"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
)

func (a *Activities) RetrieveJournalEntry(ctx context.Context, params lt.RetrieveJournalEntryParams) (*lt.RetrieveJournalEntryResult, error) {
	db, err := a.Pools.For(ctx)
	if err != nil {
		return nil, err
	}

	if params.ID <= 0 {
		return nil, internalerrors.ErrInvalidParameter{Description: "journal entry ID must be positive"}
	}

	entry := &model.JournalEntry{}
	if err := model.FindJournalEntryByIDWithRelationshipsQuery(db, params.ID).Model(entry).Scan(ctx); err != nil {
		if err == sql.ErrNoRows {
			return nil, internalerrors.ErrNotFound{ResourceType: "JournalEntry", ID: params.ID}
		}
		return nil, err
	}
	if err := model.FindPostingsByJournalEntryIDQuery(db, params.ID).Relation("LedgerAccount").Relation("Category").Scan(ctx, &entry.Postings); err != nil {
		return nil, err
	}
	err = db.NewSelect().Model(&entry.Tags).
		Join("JOIN ledger.journal_entry_tag AS jet ON jet.tag_id = tg.id").
		Where("jet.journal_entry_id = ?", params.ID).
		Apply(model.OrderTagsByName).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("load tags of journal entry: %w", err)
	}

	return &lt.RetrieveJournalEntryResult{JournalEntry: entry}, nil
}
