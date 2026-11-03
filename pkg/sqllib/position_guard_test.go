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

// Test_PositionMovementGuard checks that a movement can only be appended: it is
// inserted without an entry, or with a draft entry on the entry's own date, it
// is never changed afterwards, and it is deleted only as ADR 0011 allows. It
// runs the statements directly, as anything other than the activities would.
func (s *UnitTestSuite) Test_PositionMovementGuard() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)

	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	newEntry := func(status model.JournalEntryStatus) *model.JournalEntry {
		entry := &model.JournalEntry{OccurredAt: day, Status: status}
		if status == model.JournalEntryPosted {
			entry.Status = ""
			temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
				model.Posting{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				model.Posting{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			)
			return entry
		}
		temporaltesting.Insert(s.T(), s.DB, entry)
		return entry
	}
	// newMovement is a trade of the entry, on the given date.
	newMovement := func(entry *model.JournalEntry, occurredAt time.Time) *model.PositionMovement {
		return temporaltesting.WithMovementDefaults(&model.PositionMovement{
			JournalEntryID:     &entry.ID,
			FinancialAccountID: account.ID,
			AssetID:            asset.ID,
			UnitsDelta:         decimal.NewFromInt(5),
			OccurredAt:         occurredAt,
		})
	}
	insert := func(movement *model.PositionMovement) error {
		_, err := s.DB.NewInsert().Model(movement).Exec(s.T().Context())
		return err
	}
	// stored is a movement that is already there, for the cases that change it.
	stored := func() *model.PositionMovement {
		movement := temporaltesting.WithMovementDefaults(&model.PositionMovement{
			FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: day,
		})
		temporaltesting.Insert(s.T(), s.DB, movement)
		return movement
	}

	tests := map[string]struct {
		run func() error
		// failure is what the error has to contain, or empty when the statement has
		// to be accepted.
		failure string
	}{
		"insert a movement without an entry": {run: func() error {
			return insert(temporaltesting.WithMovementDefaults(&model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: day}))
		}},
		"insert a movement for a draft entry on its date": {run: func() error {
			return insert(newMovement(newEntry(model.JournalEntryDraft), day))
		}},
		"insert a movement for a draft entry on another date": {failure: "share the date", run: func() error {
			return insert(newMovement(newEntry(model.JournalEntryDraft), day.Add(time.Hour)))
		}},
		"insert a movement for a posted entry": {failure: "draft journal entry", run: func() error {
			return insert(newMovement(newEntry(model.JournalEntryPosted), day))
		}},
		"change the units of a movement": {failure: "append-only", run: func() error {
			movement := stored()
			_, err := s.DB.NewUpdate().Model((*model.PositionMovement)(nil)).Set("units_delta = 6").Where("id = ?", movement.ID).Exec(s.T().Context())
			return err
		}},
		"change only the description of a movement": {failure: "append-only", run: func() error {
			movement := stored()
			_, err := s.DB.NewUpdate().Model((*model.PositionMovement)(nil)).Set("description = 'edited'").Where("id = ?", movement.ID).Exec(s.T().Context())
			return err
		}},
		"delete a movement without declaring it": {failure: "append-only", run: func() error {
			movement := stored()
			_, err := s.DB.NewDelete().Model((*model.PositionMovement)(nil)).Where("id = ?", movement.ID).Exec(s.T().Context())
			return err
		}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			err := tc.run()

			if tc.failure == "" {
				require.NoError(s.T(), err)
				return
			}
			require.ErrorContains(s.T(), err, tc.failure)
		})
	}
}

// Test_PositionMovementDeleteDeclaration checks that a movement can be deleted
// only by a transaction that has declared that it is deleting the movement's
// journal entry, or, for a movement without one, its idempotency key group
// (ADR 0011), and that the declaration never allows a change.
func (s *UnitTestSuite) Test_PositionMovementDeleteDeclaration() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)

	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	postedMovement := func() *model.PositionMovement {
		entry := &model.JournalEntry{OccurredAt: day}
		movement := &model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5)}
		temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry,
			[]model.Posting{
				{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			},
			[]*model.PositionMovement{movement},
		)
		return movement
	}
	entryless := func() *model.PositionMovement {
		movement := temporaltesting.WithMovementDefaults(&model.PositionMovement{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: day})
		temporaltesting.Insert(s.T(), s.DB, movement)
		return movement
	}

	tests := map[string]struct {
		movement func() *model.PositionMovement
		// declare returns the setting and its value, or none, for the movement.
		declare func(movement *model.PositionMovement) (name, value string)
		update  bool
		failure string
	}{
		"entry movement, declares its entry": {movement: postedMovement, declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_journal_entry", fmt.Sprint(*m.JournalEntryID)
		}},
		"entry movement, declares another entry": {movement: postedMovement, failure: "append-only", declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_journal_entry", fmt.Sprint(*m.JournalEntryID + 1000)
		}},
		"entry movement, declares nothing": {movement: postedMovement, failure: "append-only", declare: func(*model.PositionMovement) (string, string) { return "", "" }},
		"entry movement, declares a group by its id": {movement: postedMovement, failure: "append-only", declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_position_movement_group", fmt.Sprint(*m.JournalEntryID)
		}},
		"entry-less movement, declares its group": {movement: entryless, declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_position_movement_group", *m.IdempotencyKey
		}},
		"entry-less movement, declares another group": {movement: entryless, failure: "append-only", declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_position_movement_group", *m.IdempotencyKey + "-other"
		}},
		"entry-less movement, declares nothing": {movement: entryless, failure: "append-only", declare: func(*model.PositionMovement) (string, string) { return "", "" }},
		"entry-less movement, declares an entry": {movement: entryless, failure: "append-only", declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_journal_entry", *m.IdempotencyKey
		}},
		"entry-less movement, declares its group and is updated": {movement: entryless, update: true, failure: "append-only", declare: func(m *model.PositionMovement) (string, string) {
			return "ivfs.deleting_position_movement_group", *m.IdempotencyKey
		}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			movement := tc.movement()

			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				if setting, value := tc.declare(movement); setting != "" {
					if _, err := tx.ExecContext(ctx, `SELECT set_config(?, ?, true)`, setting, value); err != nil {
						return err
					}
				}
				if tc.update {
					_, err := tx.NewUpdate().Model((*model.PositionMovement)(nil)).Set("units_delta = 6").Where("id = ?", movement.ID).Exec(ctx)
					return err
				}
				_, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("id = ?", movement.ID).Exec(ctx)
				return err
			})

			count, countErr := s.DB.NewSelect().Model((*model.PositionMovement)(nil)).Where("id = ?", movement.ID).Count(s.T().Context())
			require.NoError(s.T(), countErr)
			if tc.failure == "" {
				require.NoError(s.T(), err)
				require.Zero(s.T(), count)
				return
			}
			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), 1, count, "a refused statement must leave the movement in place")
		})
	}
}

// Test_PositionMovementAccountMoveDeclaration checks that a transaction that has
// declared, with ivfs.deleting_financial_account, that it is deleting a
// financial account can move that account's movements to another one, with
// nothing else changed, and delete its entry-less splits and adjustments, and
// nothing more (ADR 0012).
func (s *UnitTestSuite) Test_PositionMovementAccountMoveDeclaration() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	deleted := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	replacement := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Other account"}
	temporaltesting.Insert(s.T(), s.DB, deleted)
	temporaltesting.Insert(s.T(), s.DB, replacement)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	debit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	credit := &model.LedgerAccount{CurrencyID: eur.ID, Name: "income", Kind: model.LedgerAccountIncome}
	temporaltesting.Insert(s.T(), s.DB, debit)
	temporaltesting.Insert(s.T(), s.DB, credit)

	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	trade := func() *model.PositionMovement {
		entry := &model.JournalEntry{OccurredAt: day}
		movement := &model.PositionMovement{FinancialAccountID: deleted.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5)}
		temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry,
			[]model.Posting{
				{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			},
			[]*model.PositionMovement{movement},
		)
		return movement
	}
	entryless := func(kind model.PositionMovementKind) func() *model.PositionMovement {
		return func() *model.PositionMovement {
			movement := &model.PositionMovement{Kind: kind, FinancialAccountID: deleted.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), OccurredAt: day}
			temporaltesting.InsertPositionMovement(s.T(), s.DB, movement)
			return movement
		}
	}

	tests := map[string]struct {
		movement func() *model.PositionMovement
		// declared is the financial account the transaction declares it is
		// deleting, or 0 for none.
		declared int64
		// set is the assignment applied to the movement, or empty to delete it.
		set     string
		failure string
	}{
		"trade, declares its account and moves": {movement: trade, declared: deleted.ID, set: "financial_account_id = ?"},
		"adjustment, declares its account and moves": {
			movement: entryless(model.PositionMovementAdjustment), declared: deleted.ID, set: "financial_account_id = ?",
		},
		"trade, declares another account and moves": {
			movement: trade, declared: replacement.ID, set: "financial_account_id = ?", failure: "append-only",
		},
		"trade, declares nothing and moves": {movement: trade, set: "financial_account_id = ?", failure: "append-only"},
		"trade, declares its account and changes the units too": {
			movement: trade, declared: deleted.ID, set: "financial_account_id = ?, units_delta = 6", failure: "append-only",
		},
		"split, declares its account and is deleted":      {movement: entryless(model.PositionMovementSplit), declared: deleted.ID},
		"adjustment, declares its account and is deleted": {movement: entryless(model.PositionMovementAdjustment), declared: deleted.ID},
		"adjustment, declares another account and is deleted": {
			movement: entryless(model.PositionMovementAdjustment), declared: replacement.ID, failure: "append-only",
		},
		"transfer, declares its account and is deleted": {
			movement: entryless(model.PositionMovementTransfer), declared: deleted.ID, failure: "append-only",
		},
		"trade, declares its account and is deleted": {movement: trade, declared: deleted.ID, failure: "append-only"},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			movement := tc.movement()

			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				if tc.declared != 0 {
					if _, err := tx.ExecContext(ctx, `SELECT set_config('ivfs.deleting_financial_account', ?, true)`, fmt.Sprint(tc.declared)); err != nil {
						return err
					}
				}
				if tc.set != "" {
					_, err := tx.NewUpdate().Model((*model.PositionMovement)(nil)).Set(tc.set, replacement.ID).Where("id = ?", movement.ID).Exec(ctx)
					return err
				}
				_, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("id = ?", movement.ID).Exec(ctx)
				return err
			})

			var stored []model.PositionMovement
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("id = ?", movement.ID).Scan(s.T().Context()))
			switch {
			case tc.failure != "":
				require.ErrorContains(s.T(), err, tc.failure)
				require.Len(s.T(), stored, 1, "a refused statement must leave the movement in place")
				require.Equal(s.T(), deleted.ID, stored[0].FinancialAccountID)
				require.True(s.T(), decimal.NewFromInt(5).Equal(stored[0].UnitsDelta))
			case tc.set != "":
				require.NoError(s.T(), err)
				require.Len(s.T(), stored, 1)
				require.Equal(s.T(), replacement.ID, stored[0].FinancialAccountID)
			default:
				require.NoError(s.T(), err)
				require.Empty(s.T(), stored)
			}
		})
	}
}
