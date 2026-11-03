package posting

import (
	"context"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

type UnitTestSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	financialAccount *model.FinancialAccount
	asset            *model.Asset
	debit            *model.LedgerAccount
	credit           *model.LedgerAccount
}

func (s *UnitTestSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	s.financialAccount = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, s.financialAccount)
	s.asset = &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, s.asset)
	s.debit = &model.LedgerAccount{CurrencyID: eur.ID, Name: "cash", Kind: model.LedgerAccountAsset}
	s.credit = &model.LedgerAccount{CurrencyID: eur.ID, Name: "clearing", Kind: model.LedgerAccountAsset}
	temporaltesting.Insert(s.T(), s.DB, s.debit)
	temporaltesting.Insert(s.T(), s.DB, s.credit)
}

func (s *UnitTestSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_UnitTestSuite(t *testing.T) { suite.Run(t, new(UnitTestSuite)) }

func (s *UnitTestSuite) entry() lt.PostJournalEntryInput {
	return lt.PostJournalEntryInput{
		IdempotencyKey: "entry-with-movements",
		OccurredAt:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		Postings: []lt.PostingInput{
			{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
			{LedgerAccountID: s.credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		},
	}
}

func (s *UnitTestSuite) count(table any) int {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model(table).Count(s.T().Context())
	require.NoError(s.T(), err)
	return count
}

func (s *UnitTestSuite) Test_PostEntryWithMovementsTx() {
	input := s.entry()
	movements := []model.PositionMovement{
		{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(3)},
		{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(4)},
	}

	var entryID int64
	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) (err error) {
		entryID, err = PostEntryWithMovementsTx(ctx, tx, input, movements)
		return err
	})
	require.NoError(s.T(), err)

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", entryID).Scan(s.T().Context()))
	require.Equal(s.T(), model.JournalEntryPosted, entry.Status)

	var stored []model.PositionMovement
	require.NoError(s.T(), s.DB.NewSelect().Model(&stored).Where("journal_entry_id = ?", entryID).Order("id").Scan(s.T().Context()))
	require.Len(s.T(), stored, 2)
	for i, movement := range stored {
		require.Equal(s.T(), movements[i].ID, movement.ID, "the caller's movements are updated in place with their ids")
		require.NotZero(s.T(), movement.ID)
		require.True(s.T(), movement.OccurredAt.Equal(input.OccurredAt), "a movement takes the date of its entry")
		require.True(s.T(), movement.UnitsDelta.Equal(movements[i].UnitsDelta))
	}
}

func (s *UnitTestSuite) Test_PostEntryWithoutMovements() {
	var entryID int64
	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) (err error) {
		entryID, err = PostJournalEntryTx(ctx, tx, s.entry())
		return err
	})
	require.NoError(s.T(), err)

	require.NotZero(s.T(), entryID)
	require.Equal(s.T(), 1, s.count((*model.JournalEntry)(nil)))
	require.Zero(s.T(), s.count((*model.PositionMovement)(nil)))
}

// The entry, its postings, and its movements are one write, so a movement that
// cannot be stored has to take the entry and its postings with it.
func (s *UnitTestSuite) Test_PostEntryWithMovementsTx_IsAtomic() {
	movements := []model.PositionMovement{
		{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID, UnitsDelta: decimal.NewFromInt(3)},
		{Kind: model.PositionMovementTrade, FinancialAccountID: s.financialAccount.ID, AssetID: 999999, UnitsDelta: decimal.NewFromInt(4)},
	}

	err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
		_, err := PostEntryWithMovementsTx(ctx, tx, s.entry(), movements)
		return err
	})

	require.ErrorContains(s.T(), err, "Asset")
	require.Zero(s.T(), s.count((*model.JournalEntry)(nil)))
	require.Zero(s.T(), s.count((*model.Posting)(nil)))
	require.Zero(s.T(), s.count((*model.PositionMovement)(nil)))
}
