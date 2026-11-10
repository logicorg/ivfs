package posting

import (
	"context"
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// PostJournalEntryTx writes and posts a balanced entry using an existing SQL transaction.
func PostJournalEntryTx(ctx context.Context, tx bun.Tx, input lt.PostJournalEntryInput) (int64, error) {
	return PostEntryWithMovementsTx(ctx, tx, input, nil)
}

// PostEntryWithMovementsTx writes and posts a balanced entry together with the
// position movements that belong to it, using an existing SQL transaction. The
// movements are inserted while the entry is still a draft, and the entry is
// marked posted only afterwards, so a movement can never be attached to an
// entry that is already settled. The movements take the entry's date and are
// updated in place with their ids; the caller only supplies the rest of their
// fields.
func PostEntryWithMovementsTx(ctx context.Context, tx bun.Tx, input lt.PostJournalEntryInput, movements []model.PositionMovement) (int64, error) {
	if err := input.Validate(); err != nil {
		return 0, err
	}

	accountIDs := input.AccountIDs()

	// Sharing the account rows makes a concurrent close wait for this
	// transaction, so an account cannot be closed between the check below and
	// the insert of its postings. The fixed order avoids deadlocks.
	accounts := make([]model.LedgerAccount, 0, len(accountIDs))
	if err := tx.NewSelect().Model(&accounts).Relation("Currency").Where("la.id IN (?)", bun.List(accountIDs)).OrderExpr("la.id").For("SHARE OF la").Scan(ctx); err != nil {
		return 0, fmt.Errorf("load posting ledger accounts: %w", err)
	}
	if len(accounts) != len(accountIDs) {
		return 0, internalerrors.ErrNotFound{ResourceType: "LedgerAccount"}
	}

	for _, a := range accounts {
		if a.ClosedAt != nil {
			return 0, internalerrors.ErrResourceClosed{ResourceType: "LedgerAccount", ID: a.ID}
		}
	}

	// maps account with its currency
	accountCurrencyMap := make(map[int64]*model.Currency, len(accounts))
	for _, a := range accounts {
		accountCurrencyMap[a.ID] = a.Currency
	}

	for _, p := range input.Postings {
		currency := accountCurrencyMap[p.LedgerAccountID]
		if !p.Amount.Round(int32(currency.Precision)).Equal(p.Amount) {
			return 0, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("posting amount %s exceeds the %d decimal places of currency %s", p.Amount, currency.Precision, currency.Code)}
		}
	}

	// maps currency with its postings
	totalsPerCurrency := make(map[int64][2]decimal.Decimal)
	for _, p := range input.Postings {
		currencyID := accountCurrencyMap[p.LedgerAccountID].ID
		pair := totalsPerCurrency[currencyID]
		switch p.Side {
		case model.PostingDebit:
			pair[0] = pair[0].Add(p.Amount)
		case model.PostingCredit:
			pair[1] = pair[1].Add(p.Amount)
		}
		totalsPerCurrency[currencyID] = pair
	}

	for currencyID, pair := range totalsPerCurrency {
		if !pair[0].Equal(pair[1]) {
			return 0, internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("debits and credits do not balance for currency %d", currencyID)}
		}
	}

	entry := &model.JournalEntry{CounterpartyID: input.CounterpartyID, OccurredAt: input.OccurredAt, Description: input.Description, ExternalID: input.ExternalID, IdempotencyKey: nullableString(input.IdempotencyKey), Metadata: input.Metadata, Status: model.JournalEntryDraft}
	if _, err := tx.NewInsert().Model(entry).Returning("id").Exec(ctx); err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return 0, notFound
		}
		return 0, fmt.Errorf("insert journal entry: %w", err)
	}

	postings := make([]model.Posting, 0, len(input.Postings))
	for i, p := range input.Postings {
		postings = append(postings, model.Posting{JournalEntryID: entry.ID, LineNumber: i + 1, LedgerAccountID: p.LedgerAccountID, Amount: p.Amount, Side: p.Side})
	}

	if _, err := tx.NewInsert().Model(&postings).Exec(ctx); err != nil {
		return 0, fmt.Errorf("insert journal postings: %w", err)
	}

	if len(movements) > 0 {
		for i := range movements {
			movements[i].JournalEntryID = &entry.ID
			movements[i].OccurredAt = input.OccurredAt
		}
		if _, err := tx.NewInsert().Model(&movements).Returning("id").Exec(ctx); err != nil {
			if notFound := sqllib.ParentNotFound(err); notFound != nil {
				return 0, notFound
			}
			return 0, fmt.Errorf("insert position movements: %w", err)
		}
	}

	now := time.Now().UTC()
	entry.Status, entry.PostedAt = model.JournalEntryPosted, &now
	if _, err := tx.NewUpdate().Model(entry).Column("status", "posted_at", "updated_at").WherePK().Exec(ctx); err != nil {
		return 0, fmt.Errorf("post journal entry: %w", err)
	}

	return entry.ID, nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
