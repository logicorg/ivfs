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

// Test_PostedJournalEntryBalance checks that a posted entry can change, as long
// as the transaction that changes it leaves it with at least two postings that
// balance in each currency. Each case starts from an entry of 10 debited and 10
// credited in euros, and runs its statements directly in one transaction, as
// anything other than the activities would.
//
// The statements name their rows with :entry, :other (a second posted entry),
// :debit and :credit (the accounts of the entry), :savings (another account in
// euros) and :dollars (an account in another currency).
func (s *UnitTestSuite) Test_PostedJournalEntryBalance() {
	eur, usd := testhelpers.Euro(), testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, eur)
	temporaltesting.Insert(s.T(), s.DB, usd)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	savings := &model.LedgerAccount{CurrencyID: eur.ID, Name: "savings", Kind: model.LedgerAccountAsset}
	dollars := &model.LedgerAccount{CurrencyID: usd.ID, Name: "dollars", Kind: model.LedgerAccountAsset}
	for _, account := range []*model.LedgerAccount{debit, credit, savings, dollars} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	const (
		tooFew     = "must have at least two postings"
		unbalanced = "is not balanced by currency"
	)

	tests := []struct {
		name string
		// draft leaves the entry unposted.
		draft      bool
		statements []string
		wantErr    string
	}{
		{
			name:       "the description of a posted entry changes",
			statements: []string{`UPDATE ledger.journal_entry SET description = 'edited' WHERE id = :entry`},
		},
		{
			name:       "one amount changes",
			statements: []string{`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    unbalanced,
		},
		{
			name:       "both amounts change together",
			statements: []string{`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry`},
		},
		{
			name: "the amounts change one after the other",
			statements: []string{
				`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 1`,
				`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 2`,
			},
		},
		{
			name:       "one posting is deleted",
			statements: []string{`DELETE FROM ledger.posting WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    tooFew,
		},
		{
			name:       "every posting is deleted",
			statements: []string{`DELETE FROM ledger.posting WHERE journal_entry_id = :entry`},
			wantErr:    tooFew,
		},
		{
			name: "the entry is deleted with its postings",
			statements: []string{
				`DELETE FROM ledger.posting WHERE journal_entry_id = :entry`,
				`DELETE FROM ledger.journal_entry WHERE id = :entry`,
			},
		},
		{
			name: "the postings are replaced by balanced ones",
			statements: []string{
				`DELETE FROM ledger.posting WHERE journal_entry_id = :entry`,
				`INSERT INTO ledger.posting (journal_entry_id, line_number, ledger_account_id, amount, side)
					VALUES (:entry, 1, :savings, 7, 'debit'), (:entry, 2, :credit, 7, 'credit')`,
			},
		},
		{
			name: "a posting is added on one side only",
			statements: []string{`INSERT INTO ledger.posting (journal_entry_id, line_number, ledger_account_id, amount, side)
				VALUES (:entry, 3, :savings, 1, 'debit')`},
			wantErr: unbalanced,
		},
		{
			name:       "a posting moves to an account of the same currency",
			statements: []string{`UPDATE ledger.posting SET ledger_account_id = :savings WHERE journal_entry_id = :entry AND line_number = 1`},
		},
		{
			name:       "a posting moves to an account of another currency",
			statements: []string{`UPDATE ledger.posting SET ledger_account_id = :dollars WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    unbalanced,
		},
		{
			name:       "a posting moves to another entry",
			statements: []string{`UPDATE ledger.posting SET journal_entry_id = :other, line_number = 3 WHERE journal_entry_id = :entry AND line_number = 1`},
			wantErr:    tooFew,
		},
		{
			name: "a posted entry is reopened and then unbalanced",
			statements: []string{
				`UPDATE ledger.journal_entry SET status = 'draft' WHERE id = :entry`,
				`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 1`,
			},
		},
		{
			name:       "a draft is unbalanced",
			draft:      true,
			statements: []string{`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 1`},
		},
		{
			name:  "an unbalanced draft is posted",
			draft: true,
			statements: []string{
				`UPDATE ledger.posting SET amount = 11 WHERE journal_entry_id = :entry AND line_number = 1`,
				`UPDATE ledger.journal_entry SET status = 'posted' WHERE id = :entry`,
			},
			wantErr: unbalanced,
		},
		{
			name:       "a balanced draft is posted",
			draft:      true,
			statements: []string{`UPDATE ledger.journal_entry SET status = 'posted' WHERE id = :entry`},
		},
		{
			name:       "an entry is inserted as posted without postings",
			statements: []string{`INSERT INTO ledger.journal_entry (occurred_at, status) VALUES (now(), 'posted')`},
			wantErr:    tooFew,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			newEntry := func(status model.JournalEntryStatus) *model.JournalEntry {
				entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: status}
				err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
					if _, err := tx.NewInsert().Model(entry).Returning("id").Exec(ctx); err != nil {
						return err
					}
					postings := []model.Posting{
						{JournalEntryID: entry.ID, LineNumber: 1, LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
						{JournalEntryID: entry.ID, LineNumber: 2, LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
					}
					_, err := tx.NewInsert().Model(&postings).Exec(ctx)
					return err
				})
				require.NoError(s.T(), err)
				return entry
			}
			status := model.JournalEntryPosted
			if tt.draft {
				status = model.JournalEntryDraft
			}
			entry, other := newEntry(status), newEntry(model.JournalEntryPosted)
			// An entry goes with its postings in one transaction, or the check would
			// find it posted and without any.
			defer func() {
				ids := bun.List([]int64{entry.ID, other.ID})
				err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
					if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id IN (?)", ids).Exec(ctx); err != nil {
						return err
					}
					_, err := tx.NewDelete().Model((*model.JournalEntry)(nil)).Where("id IN (?)", ids).Exec(ctx)
					return err
				})
				require.NoError(s.T(), err)
			}()

			names := strings.NewReplacer(
				":entry", strconv.FormatInt(entry.ID, 10),
				":other", strconv.FormatInt(other.ID, 10),
				":debit", strconv.FormatInt(debit.ID, 10),
				":credit", strconv.FormatInt(credit.ID, 10),
				":savings", strconv.FormatInt(savings.ID, 10),
				":dollars", strconv.FormatInt(dollars.ID, 10),
			)

			// The check is deferred, so it is the commit that fails.
			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
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
