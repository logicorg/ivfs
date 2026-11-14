package sqllib_test

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// Test_PostedJournalEntryCategory checks that a posting of a posted entry
// carries a category only when its account has the category's kind. Each case
// starts from an uncategorized expense of 10 in euros, debited to "Expenses"
// on line 1 and credited to "cash" on line 2, and runs its statements directly
// in one transaction, as anything other than the activities would.
//
// The statements name their rows with :entry, :groceries (an expense
// category), :salary (an income category) and :income (the "Income" account).
func (s *UnitTestSuite) Test_PostedJournalEntryCategory() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	cash := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	expenses := &model.LedgerAccount{CurrencyID: eur.ID, Name: "Expenses", Kind: model.LedgerAccountExpense}
	income := &model.LedgerAccount{CurrencyID: eur.ID, Name: "Income", Kind: model.LedgerAccountIncome}
	for _, account := range []*model.LedgerAccount{cash, expenses, income} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}
	groceries := &model.Category{Kind: model.LedgerAccountExpense, Name: "Groceries"}
	salary := &model.Category{Kind: model.LedgerAccountIncome, Name: "Salary"}
	for _, category := range []*model.Category{groceries, salary} {
		temporaltesting.Insert(s.T(), s.DB, category)
	}

	const otherKind = "has a category on an account of another kind"

	tests := []struct {
		name string
		// draft leaves the entry unposted.
		draft      bool
		statements []string
		wantErr    string
	}{
		{
			name:       "the expense takes an expense category",
			statements: []string{`UPDATE ledger.posting SET category_id = :groceries WHERE journal_entry_id = :entry AND line_number = 1`},
		},
		{
			name:       "the expense takes an income category",
			statements: []string{`UPDATE ledger.posting SET category_id = :salary WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    otherKind,
		},
		{
			name:       "the posting to the balance takes a category",
			statements: []string{`UPDATE ledger.posting SET category_id = :groceries WHERE journal_entry_id = :entry AND line_number = 2`},
			wantErr:    otherKind,
		},
		{
			name: "a categorized posting moves to an account of the other kind",
			statements: []string{
				`UPDATE ledger.posting SET category_id = :groceries WHERE journal_entry_id = :entry AND line_number = 1`,
				`UPDATE ledger.posting SET ledger_account_id = :income WHERE journal_entry_id = :entry AND line_number = 1`,
			},
			wantErr: otherKind,
		},
		{
			name: "a posting moves to an account of the other kind and takes its category",
			statements: []string{
				`UPDATE ledger.posting SET ledger_account_id = :income, category_id = :salary WHERE journal_entry_id = :entry AND line_number = 1`,
			},
		},
		{
			name: "a category is set and cleared again",
			statements: []string{
				`UPDATE ledger.posting SET category_id = :salary WHERE journal_entry_id = :entry AND line_number = 1`,
				`UPDATE ledger.posting SET category_id = NULL WHERE journal_entry_id = :entry AND line_number = 1`,
			},
		},
		{
			name:       "a draft takes a category of the other kind",
			draft:      true,
			statements: []string{`UPDATE ledger.posting SET category_id = :salary WHERE journal_entry_id = :entry AND line_number = 1`},
		},
		{
			name:  "a draft with a category of the other kind is posted",
			draft: true,
			statements: []string{
				`UPDATE ledger.posting SET category_id = :salary WHERE journal_entry_id = :entry AND line_number = 1`,
				`UPDATE ledger.journal_entry SET status = 'posted' WHERE id = :entry`,
			},
			wantErr: otherKind,
		},
		{
			name:       "the posting takes a category that does not exist",
			statements: []string{`UPDATE ledger.posting SET category_id = 999999 WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    "posting_category_id_fkey",
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			status := model.JournalEntryPosted
			if tt.draft {
				status = model.JournalEntryDraft
			}
			entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: status}
			// An entry goes with its postings in one transaction, or the check would
			// find it posted and without any.
			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				if _, err := tx.NewInsert().Model(entry).Returning("id").Exec(ctx); err != nil {
					return err
				}
				postings := []model.Posting{
					{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: expenses.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					{JournalEntryID: entry.ID, LineNumber: 2, LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				}
				_, err := tx.NewInsert().Model(&postings).Exec(ctx)
				return err
			})
			require.NoError(s.T(), err)
			defer temporaltesting.DeleteJournalEntry(s.T(), s.DB, entry.ID)

			names := strings.NewReplacer(
				":entry", strconv.FormatInt(entry.ID, 10),
				":groceries", strconv.FormatInt(groceries.ID, 10),
				":salary", strconv.FormatInt(salary.ID, 10),
				":income", strconv.FormatInt(income.ID, 10),
			)

			// The check is deferred, so it is the commit that fails.
			err = s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				for _, statement := range tt.statements {
					if _, err := tx.ExecContext(ctx, names.Replace(statement)); err != nil {
						return err
					}
				}
				return nil
			})

			if tt.wantErr == "" {
				require.NoError(s.T(), err)
				return
			}
			require.ErrorContains(s.T(), err, tt.wantErr)
		})
	}
}
