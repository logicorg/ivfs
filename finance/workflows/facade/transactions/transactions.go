// Package transactions records and corrects the movements a person thinks in,
// an expense, an income, or a transfer, as one balanced journal entry each,
// deciding the sides in code (ADR 0008).
package transactions

import (
	"fmt"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	tt "github.com/logicorg/ivfs/finance/common/types/transactiontypes"
	ledgeraccountswf "github.com/logicorg/ivfs/finance/workflows/ledger/accounts"
	categorieswf "github.com/logicorg/ivfs/finance/workflows/ledger/categories"
	journalwf "github.com/logicorg/ivfs/finance/workflows/ledger/journal"
	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/workflow"
)

// movement is money leaving one ledger account and reaching another, the one
// rule the three transactions share.
type movement struct {
	// kind prefixes the idempotency key, so the same key sent for two kinds
	// of transaction cannot return the other's entry.
	kind           string
	idempotencyKey string
	from, to       int64
	// fromCategory and toCategory classify the posting of that side, which is
	// then on "Income" or "Expenses". Zero is no category.
	fromCategory, toCategory int64
	amount                   decimal.Decimal
	occurredAt               time.Time
	description              string
}

// retrieveLedgerAccounts retrieves the given ledger accounts concurrently.
func retrieveLedgerAccounts(ctx workflow.Context, ids ...int64) ([]*model.LedgerAccount, error) {
	futures := make([]workflow.ChildWorkflowFuture, len(ids))
	for i, id := range ids {
		futures[i] = workflow.ExecuteChildWorkflow(ctx, ledgeraccountswf.RetrieveLedgerAccount, lt.RetrieveLedgerAccountParams{ID: id})
	}

	accounts := make([]*model.LedgerAccount, len(ids))
	for i, future := range futures {
		var result lt.RetrieveLedgerAccountResult
		if err := future.Get(ctx, &result); err != nil {
			return nil, fmt.Errorf("retrieve ledger account %d: %w", ids[i], err)
		}
		accounts[i] = result.LedgerAccount
	}
	return accounts, nil
}

// checkBalance fails unless the account is a balance: an asset or liability
// of a financial account. The shared "Opening balances" and "FX clearing"
// accounts belong to none, so they are not balances.
func checkBalance(account *model.LedgerAccount) error {
	isBalanceKind := account.Kind == model.LedgerAccountAsset || account.Kind == model.LedgerAccountLiability
	if !isBalanceKind || account.FinancialAccountID == nil {
		return internalerrors.ErrInvalidParameter{
			Description: fmt.Sprintf("%q is not a balance: a balance is an asset or liability of a financial account", account.Name),
		}
	}
	return nil
}

// categorized checks the two things an expense and an income name, a balance
// and a category of the kind, and returns the balance with the shared account
// of that kind in the balance's currency, where the category's side is
// booked.
func categorized(ctx workflow.Context, balanceID, categoryID int64, kind model.LedgerAccountKind) (*model.LedgerAccount, int64, error) {
	accounts, err := retrieveLedgerAccounts(ctx, balanceID)
	if err != nil {
		return nil, 0, err
	}
	balance := accounts[0]
	if err := checkBalance(balance); err != nil {
		return nil, 0, err
	}

	account, err := categorieswf.CategoryAccount(ctx, categoryID, kind, balance.CurrencyID)
	if err != nil {
		return nil, 0, err
	}
	return balance, account, nil
}

// entry is the movement as one journal entry, crediting the account the money
// leaves and debiting the one it reaches.
func (m movement) entry() lt.PostJournalEntryInput {
	input := lt.PostJournalEntryInput{
		IdempotencyKey: m.kind + ":" + m.idempotencyKey,
		OccurredAt:     m.occurredAt,
		Postings: []lt.PostingInput{
			{LedgerAccountID: m.to, Amount: m.amount, Side: model.PostingDebit, CategoryID: categoryOf(m.toCategory)},
			{LedgerAccountID: m.from, Amount: m.amount, Side: model.PostingCredit, CategoryID: categoryOf(m.fromCategory)},
		},
	}
	if m.description != "" {
		input.Description = &m.description
	}
	return input
}

// categoryOf is the category of a posting, none when id is zero.
func categoryOf(id int64) *int64 {
	if id == 0 {
		return nil
	}
	return &id
}

// post posts the movement as one entry. A closed or unknown account fails
// here, inside the transaction that posts.
func post(ctx workflow.Context, m movement) (*tt.RecordResult, error) {
	var result lt.PostJournalEntryResult
	if err := workflow.ExecuteChildWorkflow(ctx, journalwf.PostJournalEntry, m.entry()).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("post %s: %w", m.kind, err)
	}
	return &tt.RecordResult{JournalEntryID: result.JournalEntryID}, nil
}

// replace makes the journal entry what entry states: its date, description,
// metadata, and postings, with no position movements. The entry keeps its own
// key, so the one in entry is not used. An unknown entry, or an account it may
// not move onto, fails here, inside the transaction that replaces.
func replace(ctx workflow.Context, journalEntryID int64, kind string, entry lt.PostJournalEntryInput) (*tt.UpdateResult, error) {
	input := lt.ReplaceJournalEntryInput{
		JournalEntryID: journalEntryID,
		OccurredAt:     entry.OccurredAt,
		Description:    entry.Description,
		Metadata:       entry.Metadata,
		Postings:       entry.Postings,
	}

	var result lt.ReplaceJournalEntryResult
	if err := workflow.ExecuteChildWorkflow(ctx, journalwf.ReplaceJournalEntry, input).Get(ctx, &result); err != nil {
		return nil, fmt.Errorf("update %s: %w", kind, err)
	}
	return &tt.UpdateResult{JournalEntryID: result.JournalEntryID, Warnings: result.Warnings}, nil
}
