package posting

import (
	"context"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func (s *UnitTestSuite) Test_RewriteEntryTx() {
	createdAt := time.Date(2026, 1, 5, 9, 30, 0, 123456000, time.UTC)
	occurredAt := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	newOccurredAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		// closedCredit closes the account the rewritten credit posting goes to.
		closedCredit bool
		// touched tells whether the entry already had a posting on that account.
		touched bool
		// unbalanced makes the rewritten debit differ from the credit.
		unbalanced bool
		wantErr    string
	}{
		{name: "the entry keeps its row and takes the new lines"},
		{name: "a closed account the entry already touched is accepted", closedCredit: true, touched: true},
		{name: "a closed account the entry never touched is refused", closedCredit: true, wantErr: "is closed"},
		{name: "unbalanced postings are refused", unbalanced: true, wantErr: "do not balance"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()

			credit := &model.LedgerAccount{CurrencyID: s.credit.CurrencyID, Name: tt.name, Kind: model.LedgerAccountAsset}
			if tt.closedCredit {
				closedAt := time.Now().UTC()
				credit.ClosedAt = &closedAt
			}
			temporaltesting.Insert(s.T(), s.DB, credit)
			counterparty := &model.Counterparty{Name: tt.name}
			temporaltesting.Insert(s.T(), s.DB, counterparty)

			description, externalID, idempotencyKey := "as recorded", "broker-"+tt.name, "rewrite-"+tt.name
			entry := &model.JournalEntry{
				OccurredAt: occurredAt, CreatedAt: createdAt, Description: &description, Metadata: map[string]string{"source": "recorded"},
				IdempotencyKey: &idempotencyKey, ExternalID: &externalID, CounterpartyID: &counterparty.ID,
			}
			oldCredit := s.credit.ID
			if tt.touched {
				oldCredit = credit.ID
			}
			oldMovement := &model.PositionMovement{FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(1)}
			temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry, []model.Posting{
				{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingDebit},
				{LedgerAccountID: oldCredit, Amount: decimal.NewFromInt(7), Side: model.PostingCredit},
			}, []*model.PositionMovement{oldMovement})
			postedAt := *entry.PostedAt

			debitAmount := decimal.NewFromInt(10)
			if tt.unbalanced {
				debitAmount = decimal.NewFromInt(11)
			}
			newDescription := "as corrected"
			// The key and the external id of the input are not the entry's, and
			// have to be ignored.
			input := lt.PostJournalEntryInput{
				IdempotencyKey: "ignored", OccurredAt: newOccurredAt, Description: &newDescription,
				Postings: []lt.PostingInput{
					{LedgerAccountID: s.debit.ID, Amount: debitAmount, Side: model.PostingDebit},
					{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				},
			}
			movements := []model.PositionMovement{
				{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(3)},
			}

			err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
				var locked model.JournalEntry
				if err := tx.NewSelect().Model(&locked).Where("je.id = ?", entry.ID).For("UPDATE").Scan(ctx); err != nil {
					return err
				}
				return RewriteEntryTx(ctx, tx, &locked, input, movements, []int64{s.debit.ID, oldCredit})
			})

			var stored model.JournalEntry
			require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Relation("Postings", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("line_number") }).Relation("PositionMovements").Where("je.id = ?", entry.ID).Scan(ctx))
			require.Len(s.T(), stored.Postings, 2)
			require.Len(s.T(), stored.PositionMovements, 1)
			movement := stored.PositionMovements[0]

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				require.True(s.T(), occurredAt.Equal(stored.OccurredAt), "the entry is left as it was")
				require.Equal(s.T(), description, *stored.Description)
				require.True(s.T(), decimal.NewFromInt(7).Equal(stored.Postings[0].Amount))
				require.Equal(s.T(), oldMovement.ID, movement.ID)
				return
			}
			require.NoError(s.T(), err)

			require.Equal(s.T(), model.JournalEntryPosted, stored.Status)
			require.True(s.T(), newOccurredAt.Equal(stored.OccurredAt))
			require.Equal(s.T(), newDescription, *stored.Description)
			require.Empty(s.T(), stored.Metadata, "the metadata of the input replaces the entry's")
			require.True(s.T(), createdAt.Equal(stored.CreatedAt), "created at %s", stored.CreatedAt)
			require.True(s.T(), postedAt.Equal(*stored.PostedAt), "posted at %s", stored.PostedAt)
			require.True(s.T(), stored.UpdatedAt.After(createdAt))
			require.Equal(s.T(), idempotencyKey, *stored.IdempotencyKey)
			require.Equal(s.T(), externalID, *stored.ExternalID)
			require.Equal(s.T(), counterparty.ID, *stored.CounterpartyID)

			require.True(s.T(), decimal.NewFromInt(10).Equal(stored.Postings[0].Amount))
			require.Equal(s.T(), credit.ID, stored.Postings[1].LedgerAccountID)

			require.Equal(s.T(), movements[0].ID, movement.ID, "the caller's movements are updated in place with their ids")
			require.NotEqual(s.T(), oldMovement.ID, movement.ID)
			require.True(s.T(), decimal.NewFromInt(3).Equal(movement.UnitsDelta))
			require.True(s.T(), createdAt.Equal(movement.CreatedAt), "a movement keeps the entry's creation time, got %s", movement.CreatedAt)
			require.Equal(s.T(), externalID, *movement.ExternalID, "a movement takes the entry's source record")
			require.True(s.T(), newOccurredAt.Equal(movement.OccurredAt))
		})
	}
}

// The old lines are deleted before the new ones are written, so a movement
// that cannot be stored has to bring the old postings back with it.
func (s *UnitTestSuite) Test_RewriteEntryTx_IsAtomic() {
	ctx := s.T().Context()
	entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)}
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, entry,
		model.Posting{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: s.credit.ID, Amount: decimal.NewFromInt(7), Side: model.PostingCredit},
	)

	movements := []model.PositionMovement{
		{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: 999999, UnitsDelta: decimal.NewFromInt(4)},
	}
	input := s.entry()
	input.Postings[0].Amount, input.Postings[1].Amount = decimal.NewFromInt(9), decimal.NewFromInt(9)

	err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return RewriteEntryTx(ctx, tx, entry, input, movements, nil)
	})

	require.ErrorContains(s.T(), err, "Asset")
	var postings []model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", entry.ID).Scan(ctx))
	require.Len(s.T(), postings, 2)
	for _, posting := range postings {
		require.True(s.T(), decimal.NewFromInt(7).Equal(posting.Amount), "the old postings are back")
	}
}
