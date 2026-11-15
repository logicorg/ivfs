package posting

import (
	"context"
	"fmt"
	"slices"
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
// entry is inserted as posted, and the database checks its balance when the
// transaction commits. The movements take the entry's date and are updated in
// place with their ids; the caller only supplies the rest of their fields.
func PostEntryWithMovementsTx(ctx context.Context, tx bun.Tx, input lt.PostJournalEntryInput, movements []model.PositionMovement) (int64, error) {
	if err := checkPostingsTx(ctx, tx, input, nil); err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	entry := &model.JournalEntry{CounterpartyID: input.CounterpartyID, OccurredAt: input.OccurredAt, Description: input.Description, ExternalID: input.ExternalID, IdempotencyKey: nullableString(input.IdempotencyKey), Metadata: input.Metadata, Status: model.JournalEntryPosted, PostedAt: &now}
	if _, err := tx.NewInsert().Model(entry).Returning("id").Exec(ctx); err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return 0, notFound
		}
		return 0, fmt.Errorf("insert journal entry: %w", err)
	}

	if err := insertLinesTx(ctx, tx, entry, input.Postings, movements); err != nil {
		return 0, err
	}
	return entry.ID, nil
}

// checkPostingsTx checks what an entry needs of its postings before they are
// written: accounts that exist and are open, amounts within the precision of
// their currency, debits that equal credits in each currency, and categories
// a posting may carry. A closed ledger account is accepted when it is one of
// allowedClosedLedgerAccountIDs.
func checkPostingsTx(ctx context.Context, tx bun.Tx, input lt.PostJournalEntryInput, allowedClosedLedgerAccountIDs []int64) error {
	if err := input.Validate(); err != nil {
		return err
	}

	accountIDs := input.AccountIDs()

	// Sharing the account rows makes a concurrent close wait for this
	// transaction, so an account cannot be closed between the check below and
	// the insert of its postings. The fixed order avoids deadlocks.
	accounts := make([]model.LedgerAccount, 0, len(accountIDs))
	if err := tx.NewSelect().Model(&accounts).Relation("Currency").Where("la.id IN (?)", bun.List(accountIDs)).OrderExpr("la.id").For("SHARE OF la").Scan(ctx); err != nil {
		return fmt.Errorf("load posting ledger accounts: %w", err)
	}
	if len(accounts) != len(accountIDs) {
		return internalerrors.ErrNotFound{ResourceType: "LedgerAccount"}
	}

	for _, a := range accounts {
		if a.ClosedAt != nil && !slices.Contains(allowedClosedLedgerAccountIDs, a.ID) {
			return internalerrors.ErrResourceClosed{ResourceType: "LedgerAccount", ID: a.ID}
		}
	}

	// maps account with its currency
	accountCurrencyMap := make(map[int64]*model.Currency, len(accounts))
	accountKinds := make(map[int64]model.LedgerAccountKind, len(accounts))
	for _, a := range accounts {
		accountCurrencyMap[a.ID] = a.Currency
		accountKinds[a.ID] = a.Kind
	}

	if err := checkCategoriesTx(ctx, tx, input, accountKinds); err != nil {
		return err
	}

	for _, p := range input.Postings {
		currency := accountCurrencyMap[p.LedgerAccountID]
		if !p.Amount.Round(int32(currency.Precision)).Equal(p.Amount) {
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("posting amount %s exceeds the %d decimal places of currency %s", p.Amount, currency.Precision, currency.Code)}
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
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("debits and credits do not balance for currency %d", currencyID)}
		}
	}
	return nil
}

// checkCategoriesTx checks the categories the postings carry: each exists, has
// no children, and has the kind of its posting's account. The database checks
// the kind too, but only at commit and without saying which posting is wrong.
//
// A closed category is accepted. Closing one only stops the UI from offering
// it: a category has no balance and stands for nothing that can cease to
// exist, and an expense that arrives late still belongs where it belongs.
func checkCategoriesTx(ctx context.Context, tx bun.Tx, input lt.PostJournalEntryInput, accountKinds map[int64]model.LedgerAccountKind) error {
	categoryIDs := input.CategoryIDs()
	if len(categoryIDs) == 0 {
		return nil
	}

	// Sharing the category rows makes a concurrent delete or new child wait for
	// this transaction, since each of them takes a stronger lock on the
	// category before it looks for postings. The categories come after the
	// accounts, in a fixed order, to avoid deadlocks.
	categories := make([]model.Category, 0, len(categoryIDs))
	if err := tx.NewSelect().Model(&categories).Where("cat.id IN (?)", bun.List(categoryIDs)).OrderExpr("cat.id").For("SHARE OF cat").Scan(ctx); err != nil {
		return fmt.Errorf("load posting categories: %w", err)
	}
	if len(categories) != len(categoryIDs) {
		return internalerrors.ErrNotFound{ResourceType: "Category"}
	}

	categoryKinds := make(map[int64]model.LedgerAccountKind, len(categories))
	for _, c := range categories {
		categoryKinds[c.ID] = c.Kind
	}

	var parentIDs []int64
	if err := tx.NewSelect().Model((*model.Category)(nil)).ColumnExpr("DISTINCT cat.parent_id").Where("cat.parent_id IN (?)", bun.List(categoryIDs)).OrderExpr("cat.parent_id").Scan(ctx, &parentIDs); err != nil {
		return fmt.Errorf("load children of posting categories: %w", err)
	}
	if len(parentIDs) > 0 {
		return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("category %d has children, and a posting takes one of them", parentIDs[0])}
	}

	for _, p := range input.Postings {
		if p.CategoryID == nil {
			continue
		}
		if accountKind := accountKinds[p.LedgerAccountID]; categoryKinds[*p.CategoryID] != accountKind {
			return internalerrors.ErrInvalidParameter{Description: fmt.Sprintf("category %d is of kind %s, and ledger account %d of kind %s", *p.CategoryID, categoryKinds[*p.CategoryID], p.LedgerAccountID, accountKind)}
		}
	}
	return nil
}

// insertLinesTx inserts the postings and the movements of an entry that is
// already stored. The movements take the entry's id and date.
func insertLinesTx(ctx context.Context, tx bun.Tx, entry *model.JournalEntry, lines []lt.PostingInput, movements []model.PositionMovement) error {
	postings := make([]model.Posting, 0, len(lines))
	for i, p := range lines {
		postings = append(postings, model.Posting{JournalEntryID: entry.ID, LineNumber: i + 1, LedgerAccountID: p.LedgerAccountID, Amount: p.Amount, Side: p.Side, CategoryID: p.CategoryID})
	}
	if _, err := tx.NewInsert().Model(&postings).Exec(ctx); err != nil {
		return fmt.Errorf("insert journal postings: %w", err)
	}

	if len(movements) == 0 {
		return nil
	}
	for i := range movements {
		movements[i].JournalEntryID = &entry.ID
		movements[i].OccurredAt = entry.OccurredAt
	}
	if _, err := tx.NewInsert().Model(&movements).Returning("id").Exec(ctx); err != nil {
		if notFound := sqllib.ParentNotFound(err); notFound != nil {
			return notFound
		}
		return fmt.Errorf("insert position movements: %w", err)
	}
	return nil
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
