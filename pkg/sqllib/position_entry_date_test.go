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

// Test_PositionMovementEntryDate checks that a movement of a journal entry is
// on the entry's date, when it is written and whenever either date changes
// afterwards, and that a movement without an entry is on any date. Each case
// starts from a posted entry on 1 March at noon with one movement, and runs its
// statement directly, as anything other than the activities would.
//
// The statements name their rows with :entry, :account and :asset.
func (s *UnitTestSuite) Test_PositionMovementEntryDate() {
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
	const entryDate = "position_movement_entry_date_fkey"

	tests := []struct {
		name      string
		statement string
		wantErr   string
		// wantMovementAt is the date the entry's first movement ends on, when the
		// statement is accepted.
		wantMovementAt time.Time
	}{
		{
			name: "a movement is added to a posted entry on its date",
			statement: `INSERT INTO asset.position_movement (journal_entry_id, kind, financial_account_id, asset_id, units_delta, occurred_at)
				VALUES (:entry, 'trade', :account, :asset, 1, '2026-03-01 12:00:00+00')`,
			wantMovementAt: day,
		},
		{
			name: "a movement is added to an entry on another date",
			statement: `INSERT INTO asset.position_movement (journal_entry_id, kind, financial_account_id, asset_id, units_delta, occurred_at)
				VALUES (:entry, 'trade', :account, :asset, 1, '2026-03-01 13:00:00+00')`,
			wantErr: entryDate,
		},
		{
			name:      "the date of a movement changes alone",
			statement: `UPDATE asset.position_movement SET occurred_at = '2026-03-02 12:00:00+00' WHERE journal_entry_id = :entry`,
			wantErr:   entryDate,
		},
		{
			name:           "the date of the entry changes",
			statement:      `UPDATE ledger.journal_entry SET occurred_at = '2026-03-02 12:00:00+00' WHERE id = :entry`,
			wantMovementAt: day.AddDate(0, 0, 1),
		},
		{
			name:           "the units of a movement change",
			statement:      `UPDATE asset.position_movement SET units_delta = 6 WHERE journal_entry_id = :entry`,
			wantMovementAt: day,
		},
		{
			name: "a movement without an entry is on any date",
			statement: `INSERT INTO asset.position_movement (kind, financial_account_id, asset_id, units_delta, occurred_at, idempotency_key, line_number)
				VALUES ('adjustment', :account, :asset, 1, '2020-01-01 00:00:00+00', 'entry-less on any date', 1)`,
			wantMovementAt: day,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			entry := &model.JournalEntry{OccurredAt: day}
			temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry,
				[]model.Posting{
					{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				},
				[]*model.PositionMovement{{FinancialAccountID: account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5)}},
			)
			defer func() {
				err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
					if _, err := tx.NewDelete().Model((*model.PositionMovement)(nil)).Where("financial_account_id = ?", account.ID).Exec(ctx); err != nil {
						return err
					}
					if _, err := tx.NewDelete().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", entry.ID).Exec(ctx); err != nil {
						return err
					}
					_, err := tx.NewDelete().Model((*model.JournalEntry)(nil)).Where("id = ?", entry.ID).Exec(ctx)
					return err
				})
				require.NoError(s.T(), err)
			}()

			names := strings.NewReplacer(
				":entry", strconv.FormatInt(entry.ID, 10),
				":account", strconv.FormatInt(account.ID, 10),
				":asset", strconv.FormatInt(asset.ID, 10),
			)

			_, err := s.DB.ExecContext(s.T().Context(), names.Replace(tt.statement))

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				return
			}
			require.NoError(s.T(), err)

			var movement model.PositionMovement
			require.NoError(s.T(), s.DB.NewSelect().Model(&movement).Where("pm.journal_entry_id = ?", entry.ID).Order("pm.id").Limit(1).Scan(s.T().Context()))
			require.True(s.T(), tt.wantMovementAt.Equal(movement.OccurredAt), "movement on %s", movement.OccurredAt)
		})
	}
}
