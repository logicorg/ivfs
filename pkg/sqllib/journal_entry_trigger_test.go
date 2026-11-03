package sqllib_test

import (
	"context"
	"fmt"
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

// Test_JournalEntryImmutability checks that only a draft entry can change or be
// deleted, and that posting or voiding it is the only way it ever leaves the
// draft state. It runs the statements directly, as anything other than the
// activities would.
func (s *UnitTestSuite) Test_JournalEntryImmutability() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)

	newEntry := func(status model.JournalEntryStatus) *model.JournalEntry {
		entry := &model.JournalEntry{OccurredAt: time.Now().UTC()}
		if status == model.JournalEntryPosted {
			temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
				model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			)
			return entry
		}
		entry.Status = status
		temporaltesting.Insert(s.T(), s.DB, entry)
		return entry
	}

	update := func(assignment string, args ...any) func(ctx context.Context, id int64) error {
		return func(ctx context.Context, id int64) error {
			_, err := s.DB.NewUpdate().Model((*model.JournalEntry)(nil)).Set(assignment, args...).Where("id = ?", id).Exec(ctx)
			return err
		}
	}
	remove := func(ctx context.Context, id int64) error {
		_, err := s.DB.NewDelete().Model((*model.JournalEntry)(nil)).Where("id = ?", id).Exec(ctx)
		return err
	}

	tests := map[string]struct {
		status    model.JournalEntryStatus
		operation func(ctx context.Context, id int64) error
		expErr    bool
	}{
		"edit a draft":          {status: model.JournalEntryDraft, operation: update("description = ?", "edited")},
		"delete a draft":        {status: model.JournalEntryDraft, operation: remove},
		"edit a posted entry":   {status: model.JournalEntryPosted, operation: update("description = ?", "edited"), expErr: true},
		"reopen a posted entry": {status: model.JournalEntryPosted, operation: update("status = 'draft'"), expErr: true},
		"delete a posted entry": {status: model.JournalEntryPosted, operation: remove, expErr: true},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			entry := newEntry(tc.status)

			err := tc.operation(s.T().Context(), entry.ID)
			if !tc.expErr {
				require.NoError(s.T(), err)
				return
			}

			require.ErrorContains(s.T(), err, "cannot be changed or deleted")

			var stored model.JournalEntry
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", entry.ID).Scan(s.T().Context()), "a refused change must leave the entry in place")
			require.Equal(s.T(), tc.status, stored.Status)
			require.Nil(s.T(), stored.Description)
		})
	}
}

// Test_JournalEntryDeleteDeclaration checks that a posted entry and its postings
// can be deleted only by a transaction that has declared, with
// ivfs.deleting_journal_entry, that it is deleting that entry (ADR 0011), and
// that the declaration of another entry, or none, changes nothing.
func (s *UnitTestSuite) Test_JournalEntryDeleteDeclaration() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)

	tests := map[string]struct {
		// postingsSetting and entrySetting are what the transaction declares before
		// deleting the postings and the entry, given the id of the entry and of
		// another one. An empty setting is not declared at all.
		postingsSetting, entrySetting func(entryID, otherID int64) string
		failure                       string
	}{
		"declares the entry": {
			postingsSetting: func(entryID, _ int64) string { return fmt.Sprint(entryID) },
			entrySetting:    func(entryID, _ int64) string { return fmt.Sprint(entryID) },
		},
		"declares another entry": {
			postingsSetting: func(_, otherID int64) string { return fmt.Sprint(otherID) },
			entrySetting:    func(_, otherID int64) string { return fmt.Sprint(otherID) },
			failure:         "immutable",
		},
		"declares nothing": {
			postingsSetting: func(_, _ int64) string { return "" },
			entrySetting:    func(_, _ int64) string { return "" },
			failure:         "immutable",
		},
		"declares the entry for its postings only": {
			postingsSetting: func(entryID, _ int64) string { return fmt.Sprint(entryID) },
			entrySetting:    func(_, otherID int64) string { return fmt.Sprint(otherID) },
			failure:         "cannot be changed or deleted",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			entry := &model.JournalEntry{OccurredAt: time.Now().UTC()}
			temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
				model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			)
			other := &model.JournalEntry{OccurredAt: time.Now().UTC()}
			temporaltesting.Insert(s.T(), s.DB, other)

			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				declare := func(value string) error {
					if value == "" {
						return nil
					}
					_, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_journal_entry', ?, true)`, value)
					return err
				}
				if err := declare(tc.postingsSetting(entry.ID, other.ID)); err != nil {
					return err
				}
				if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", entry.ID).Exec(ctx); err != nil {
					return err
				}
				if err := declare(tc.entrySetting(entry.ID, other.ID)); err != nil {
					return err
				}
				_, err := tx.NewDelete().Model((*model.JournalEntry)(nil)).Where("id = ?", entry.ID).Exec(ctx)
				return err
			})

			count, countErr := s.DB.NewSelect().Model((*model.JournalEntry)(nil)).Where("id = ?", entry.ID).Count(s.T().Context())
			require.NoError(s.T(), countErr)
			if tc.failure == "" {
				require.NoError(s.T(), err)
				require.Zero(s.T(), count)
				return
			}
			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), 1, count, "a refused delete must leave the entry in place")
		})
	}
}

// Test_JournalEntryUpdateIgnoresDeleteDeclaration checks that declaring the
// deletion of an entry does not let it be updated.
func (s *UnitTestSuite) Test_JournalEntryUpdateIgnoresDeleteDeclaration() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC()}
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
		model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	)

	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_journal_entry', ?, true)`, fmt.Sprint(entry.ID)); err != nil {
			return err
		}
		_, err := tx.NewUpdate().Model((*model.JournalEntry)(nil)).Set("description = 'edited'").Where("id = ?", entry.ID).Exec(ctx)
		return err
	})

	require.ErrorContains(s.T(), err, "cannot be changed or deleted")
}

// Test_PostingAccountMoveDeclaration checks that a posting of a posted entry can
// move to another ledger account only in a transaction that has declared, with
// ivfs.deleting_ledger_account, that it is deleting the posting's account, only
// onto an account of the same currency, and with nothing else changed (ADR 0012).
func (s *UnitTestSuite) Test_PostingAccountMoveDeclaration() {
	eur, usd := testhelpers.Euro(), testhelpers.USDollar()
	temporaltesting.Insert(s.T(), s.DB, eur)
	temporaltesting.Insert(s.T(), s.DB, usd)
	deleted := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	replacement := &model.LedgerAccount{CurrencyID: eur.ID, Name: "bank", Kind: model.LedgerAccountAsset}
	foreign := &model.LedgerAccount{CurrencyID: usd.ID, Name: "dollars", Kind: model.LedgerAccountAsset}
	for _, account := range []*model.LedgerAccount{deleted, credit, replacement, foreign} {
		temporaltesting.Insert(s.T(), s.DB, account)
	}

	tests := map[string]struct {
		// declared is the account the transaction declares it is deleting, or 0 for
		// none.
		declared int64
		// set is the assignment applied to the posting on the deleted account.
		set     string
		args    []any
		failure string
	}{
		"declares the account and moves the posting": {
			declared: deleted.ID, set: "ledger_account_id = ?", args: []any{replacement.ID},
		},
		"declares another account": {
			declared: credit.ID, set: "ledger_account_id = ?", args: []any{replacement.ID}, failure: "immutable",
		},
		"declares nothing": {
			set: "ledger_account_id = ?", args: []any{replacement.ID}, failure: "immutable",
		},
		"declares the account and changes the amount too": {
			declared: deleted.ID, set: "ledger_account_id = ?, amount = 11", args: []any{replacement.ID}, failure: "immutable",
		},
		"declares the account and moves across currencies": {
			declared: deleted.ID, set: "ledger_account_id = ?", args: []any{foreign.ID}, failure: "different currencies",
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			entry := &model.JournalEntry{OccurredAt: time.Now().UTC()}
			temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
				model.Posting{LedgerAccountID: deleted.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			)

			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				if tc.declared != 0 {
					if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_ledger_account', ?, true)`, fmt.Sprint(tc.declared)); err != nil {
						return err
					}
				}
				_, err := tx.NewUpdate().Model((*model.Posting)(nil)).Set(tc.set, tc.args...).
					Where("journal_entry_id = ? AND ledger_account_id = ?", entry.ID, deleted.ID).Exec(ctx)
				return err
			})

			var postings []model.Posting
			require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", entry.ID).Order("line_number").Scan(s.T().Context()))
			if tc.failure == "" {
				require.NoError(s.T(), err)
				require.Equal(s.T(), replacement.ID, postings[0].LedgerAccountID)
				return
			}
			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), deleted.ID, postings[0].LedgerAccountID, "a refused move must leave the posting in place")
			require.True(s.T(), decimal.NewFromInt(10).Equal(postings[0].Amount))
		})
	}
}

// Test_PostingDeleteIgnoresAccountDeclaration checks that declaring the deletion
// of a ledger account does not let a posting of a posted entry on it be deleted.
func (s *UnitTestSuite) Test_PostingDeleteIgnoresAccountDeclaration() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC()}
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
		model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	)

	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_ledger_account', ?, true)`, fmt.Sprint(debit.ID)); err != nil {
			return err
		}
		_, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("ledger_account_id = ?", debit.ID).Exec(ctx)
		return err
	})

	require.ErrorContains(s.T(), err, "immutable")
}
