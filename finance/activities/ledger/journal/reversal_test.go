package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func (s *UnitTestSuite) TestReverseJournalEntryValidation() {
	_, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{IdempotencyKey: "x"})
	require.Error(s.T(), err)
	_, err = s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: 1})
	require.Error(s.T(), err)
}

func (s *UnitTestSuite) TestReverseJournalEntry() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)

	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "asset", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "income", model.LedgerAccountIncome)

	posted, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{
		IdempotencyKey: "original-entry", OccurredAt: time.Now().UTC(),
		Postings: []lt.PostingInput{{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingDebit}, {LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingCredit}},
	})

	require.NoError(s.T(), err)
	var original lt.PostJournalEntryResult
	require.NoError(s.T(), posted.Get(&original))

	reversed, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: original.JournalEntryID, IdempotencyKey: "reverse-entry"})
	require.NoError(s.T(), err)
	var result lt.ReverseJournalEntryResult
	require.NoError(s.T(), reversed.Get(&result))

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.Equal(s.T(), &original.JournalEntryID, entry.ReversalOfEntryID)
	var postings []model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", result.JournalEntryID).Order("line_number").Scan(s.T().Context()))
	require.Len(s.T(), postings, 2)
	require.Equal(s.T(), model.PostingCredit, postings[0].Side)
	require.Equal(s.T(), model.PostingDebit, postings[1].Side)
}

// TestReverseJournalEntry_InvertsEveryMovement checks that all the units
// movements of an entry are inverted, with their cost basis, and that the
// reversal does not inherit the external ID of the entry it reverses.
func (s *UnitTestSuite) TestReverseJournalEntry_InvertsEveryMovement() {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	debit := testLedgerAccount(s.T(), s.DB, euro.ID, "asset", model.LedgerAccountAsset)
	credit := testLedgerAccount(s.T(), s.DB, euro.ID, "income", model.LedgerAccountIncome)

	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	first := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker A"}
	second := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker B"}
	temporaltesting.Insert(s.T(), s.DB, first)
	temporaltesting.Insert(s.T(), s.DB, second)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	original := &model.JournalEntry{OccurredAt: time.Now().UTC(), ExternalID: ptr.NewPtr("provider-7")}
	firstMovement := &model.PositionMovement{FinancialAccountID: first.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(3), CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(30)), CostBasisCurrencyID: ptr.Int64(euro.ID), ExternalID: ptr.NewPtr("provider-7")}
	secondMovement := &model.PositionMovement{FinancialAccountID: second.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(4)}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, original,
		[]model.Posting{
			{LedgerAccountID: debit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingDebit},
			{LedgerAccountID: credit.ID, Amount: decimal.NewFromInt(25), Side: model.PostingCredit},
		},
		[]*model.PositionMovement{firstMovement, secondMovement},
	)

	input := lt.ReverseJournalEntryInput{JournalEntryID: original.ID, IdempotencyKey: "reverse-two-movements"}
	reversed, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, input)
	require.NoError(s.T(), err)
	var result lt.ReverseJournalEntryResult
	require.NoError(s.T(), reversed.Get(&result))

	var entry model.JournalEntry
	require.NoError(s.T(), s.DB.NewSelect().Model(&entry).Where("id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.Nil(s.T(), entry.ExternalID)

	var inverted []model.PositionMovement
	require.NoError(s.T(), s.DB.NewSelect().Model(&inverted).Where("journal_entry_id = ?", result.JournalEntryID).Order("financial_account_id").Scan(s.T().Context()))
	require.Len(s.T(), inverted, 2)

	require.Equal(s.T(), first.ID, inverted[0].FinancialAccountID)
	require.Equal(s.T(), model.PositionMovementTrade, inverted[0].Kind)
	require.Equal(s.T(), &firstMovement.ID, inverted[0].ReversalOfMovementID, "the inverse refers to the movement it compensates")
	require.Equal(s.T(), ptr.Int64(euro.ID), inverted[0].CostBasisCurrencyID)
	require.True(s.T(), inverted[0].UnitsDelta.Equal(decimal.NewFromInt(-3)))
	require.NotNil(s.T(), inverted[0].CostBasisAmount)
	require.True(s.T(), inverted[0].CostBasisAmount.Equal(decimal.NewFromInt(-30)))
	require.Nil(s.T(), inverted[0].ExternalID)

	require.Equal(s.T(), second.ID, inverted[1].FinancialAccountID)
	require.Equal(s.T(), &secondMovement.ID, inverted[1].ReversalOfMovementID)
	require.True(s.T(), inverted[1].UnitsDelta.Equal(decimal.NewFromInt(-4)))
	require.Nil(s.T(), inverted[1].CostBasisAmount)

	// Retrying returns the same reversal without inverting the movements again.
	retried, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, input)
	require.NoError(s.T(), err)
	var retryResult lt.ReverseJournalEntryResult
	require.NoError(s.T(), retried.Get(&retryResult))
	require.Equal(s.T(), result.JournalEntryID, retryResult.JournalEntryID)
	require.Empty(s.T(), result.Warnings)
	require.Empty(s.T(), retryResult.Warnings)
	count, err := s.DB.NewSelect().Model((*model.PositionMovement)(nil)).Where("journal_entry_id = ?", result.JournalEntryID).Count(s.T().Context())
	require.NoError(s.T(), err)
	require.EqualValues(s.T(), 2, count)
}

// held is a financial account holding an asset, with the ledger accounts that
// trades of it are booked against.
type held struct {
	account *model.FinancialAccount
	asset   *model.Asset
	debit   *model.LedgerAccount
	credit  *model.LedgerAccount
}

func (s *UnitTestSuite) newHeld() held {
	euro := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, euro)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)

	return held{
		account: account,
		asset:   asset,
		debit:   testLedgerAccount(s.T(), s.DB, euro.ID, "cash", model.LedgerAccountAsset),
		credit:  testLedgerAccount(s.T(), s.DB, euro.ID, "clearing", model.LedgerAccountAsset),
	}
}

// insertTrade inserts a posted entry with a movement of units of the asset.
func (s *UnitTestSuite) insertTrade(h held, units int64, at time.Time) *model.JournalEntry {
	entry := &model.JournalEntry{OccurredAt: at}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry,
		[]model.Posting{
			{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
			{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
		},
		[]*model.PositionMovement{{FinancialAccountID: h.account.ID, AssetID: h.asset.ID, UnitsDelta: decimal.NewFromInt(units)}},
	)
	return entry
}

// A reversal gives units back, so when they were sold since it is written
// anyway and warns about the position it leaves negative, at the date of the
// reversal. Undoing the sale afterwards gives the units back, but the position
// was still negative between the two reversals, so it still warns.
func (s *UnitTestSuite) TestReverseJournalEntry_UnitsSoldSince() {
	h := s.newHeld()
	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	purchase := s.insertTrade(h, 10, day)
	sale := s.insertTrade(h, -10, day.AddDate(0, 0, 1))

	reversed, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: purchase.ID, IdempotencyKey: "reverse-purchase"})
	require.NoError(s.T(), err)
	var result lt.ReverseJournalEntryResult
	require.NoError(s.T(), reversed.Get(&result))
	require.Equal(s.T(), 3, s.Count[model.JournalEntry]())
	require.Len(s.T(), result.Warnings, 1)
	warning := result.Warnings[0]
	require.Equal(s.T(), h.account.ID, warning.FinancialAccountID)
	require.Equal(s.T(), h.account.Name, warning.FinancialAccountName)
	require.Equal(s.T(), h.asset.ID, warning.AssetID)
	require.Equal(s.T(), h.asset.Name, warning.AssetName)
	require.True(s.T(), warning.Units.Equal(decimal.NewFromInt(-10)), "holds %s units", warning.Units)
	require.True(s.T(), warning.At.After(day.AddDate(0, 0, 1)), "the reversal is dated when it is made")

	// Undoing the sale gives the units back, after the position went negative.
	undone, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: sale.ID, IdempotencyKey: "reverse-sale"})
	require.NoError(s.T(), err)
	var undoResult lt.ReverseJournalEntryResult
	require.NoError(s.T(), undone.Get(&undoResult))
	require.Len(s.T(), undoResult.Warnings, 1)
	require.Equal(s.T(), result.Warnings[0], undoResult.Warnings[0])
	require.Equal(s.T(), 4, s.Count[model.JournalEntry]())
}

// A retry returns the warnings as they are now, so none once the units that the
// sale needed have been backfilled.
func (s *UnitTestSuite) TestReverseJournalEntry_IdempotentWarnings() {
	h := s.newHeld()
	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	purchase := s.insertTrade(h, 10, day.AddDate(0, 0, 1))
	s.insertTrade(h, -10, day.AddDate(0, 0, 2))
	input := lt.ReverseJournalEntryInput{JournalEntryID: purchase.ID, IdempotencyKey: "reverse-purchase"}

	first, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, input)
	require.NoError(s.T(), err)
	var created lt.ReverseJournalEntryResult
	require.NoError(s.T(), first.Get(&created))
	require.Len(s.T(), created.Warnings, 1)

	second, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, input)
	require.NoError(s.T(), err)
	var retried lt.ReverseJournalEntryResult
	require.NoError(s.T(), second.Get(&retried))
	require.Equal(s.T(), created, retried)
	require.Equal(s.T(), 3, s.Count[model.JournalEntry]())

	s.insertTrade(h, 10, day)

	third, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, input)
	require.NoError(s.T(), err)
	var afterBackfill lt.ReverseJournalEntryResult
	require.NoError(s.T(), third.Get(&afterBackfill))
	require.Equal(s.T(), created.JournalEntryID, afterBackfill.JournalEntryID)
	require.Empty(s.T(), afterBackfill.Warnings)
	require.Equal(s.T(), 4, s.Count[model.JournalEntry]())
}

// Closing is final, so an entry made while the account was open has to remain
// reversible.
func (s *UnitTestSuite) TestReverseJournalEntry_ClosedFinancialAccount() {
	h := s.newHeld()
	purchase := s.insertTrade(h, 10, time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC))

	closedAt := time.Now().UTC()
	h.account.ClosedAt = &closedAt
	_, err := s.DB.NewUpdate().Model(h.account).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: purchase.ID, IdempotencyKey: "reverse-on-closed"})

	require.NoError(s.T(), err)
}
