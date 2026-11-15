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
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())
	require.Zero(s.T(), s.Count[model.PositionMovement]())
}

// Test_PostEntryCategories checks which category a posting may carry. Each
// case posts an expense of 10, debited to an expense account and credited to
// cash, with a category of its own on one of the two postings.
func (s *UnitTestSuite) Test_PostEntryCategories() {
	tests := []struct {
		name string
		// kind is the kind of the category; no category is created when empty.
		kind model.LedgerAccountKind
		// closed closes the category, and parent gives it a child.
		closed, parent bool
		// onCash puts the category on the credit to cash, not on the expense.
		onCash bool
		// unknownID is a category id that is not a category's.
		unknownID int64
		wantErr   string
	}{
		{name: "no category"},
		{name: "an expense category on the expense", kind: model.LedgerAccountExpense},
		{name: "an income category on the expense", kind: model.LedgerAccountIncome, wantErr: "is of kind income"},
		{name: "an expense category on the cash", kind: model.LedgerAccountExpense, onCash: true, wantErr: "of kind asset"},
		{name: "a closed category is accepted", kind: model.LedgerAccountExpense, closed: true},
		{name: "a category with children", kind: model.LedgerAccountExpense, parent: true, wantErr: "has children"},
		{name: "a category that does not exist", unknownID: 999999, wantErr: "Category not found"},
		{name: "a category id that is not one", unknownID: -1, wantErr: "invalid posting category"},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := s.T().Context()

			expenses := &model.LedgerAccount{CurrencyID: s.debit.CurrencyID, Name: tt.name, Kind: model.LedgerAccountExpense}
			temporaltesting.Insert(s.T(), s.DB, expenses)

			var categoryID *int64
			if tt.unknownID != 0 {
				categoryID = &tt.unknownID
			}
			if tt.kind != "" {
				category := &model.Category{Kind: tt.kind, Name: tt.name}
				if tt.closed {
					closedAt := time.Now().UTC()
					category.ClosedAt = &closedAt
				}
				temporaltesting.Insert(s.T(), s.DB, category)
				if tt.parent {
					temporaltesting.Insert(s.T(), s.DB, &model.Category{ParentID: &category.ID, Kind: tt.kind, Name: "child"})
				}
				categoryID = &category.ID
			}

			input := lt.PostJournalEntryInput{
				IdempotencyKey: tt.name,
				OccurredAt:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
				Postings: []lt.PostingInput{
					{LedgerAccountID: expenses.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
					{LedgerAccountID: s.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
				},
			}
			categorized := 0
			if tt.onCash {
				categorized = 1
			}
			input.Postings[categorized].CategoryID = categoryID

			var entryID int64
			err := s.DB.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) (err error) {
				entryID, err = PostJournalEntryTx(ctx, tx, input)
				return err
			})

			if tt.wantErr != "" {
				require.ErrorContains(s.T(), err, tt.wantErr)
				require.Zero(s.T(), entryID)
				return
			}
			require.NoError(s.T(), err)

			var postings []model.Posting
			require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", entryID).Order("line_number").Scan(ctx))
			require.Len(s.T(), postings, 2)
			require.Equal(s.T(), categoryID, postings[0].CategoryID)
			require.Nil(s.T(), postings[1].CategoryID)
		})
	}
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
	require.Zero(s.T(), s.Count[model.JournalEntry]())
	require.Zero(s.T(), s.Count[model.Posting]())
	require.Zero(s.T(), s.Count[model.PositionMovement]())
}
