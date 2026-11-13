package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

var replaceDay = time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)

// replaceInput is a correction of an entry to amount euro between the ledger
// accounts of h, on replaceDay, moving units of the asset when units is not
// zero.
func replaceInput(h held, entryID int64, asset *model.Asset, amount, units int64) lt.ReplaceJournalEntryInput {
	input := lt.ReplaceJournalEntryInput{
		JournalEntryID: entryID,
		OccurredAt:     replaceDay,
		Description:    ptr.NewPtr("as corrected"),
		Postings: []lt.PostingInput{
			{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit},
			{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit},
		},
	}
	if units != 0 {
		input.Movements = []lt.ReplaceMovementInput{{Kind: model.PositionMovementTrade, FinancialAccountID: h.account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(units)}}
	}
	return input
}

// storedEntry reads an entry with its postings, by line, and its movements.
func (s *UnitTestSuite) storedEntry(id int64) model.JournalEntry {
	s.T().Helper()

	var entry model.JournalEntry
	err := s.DB.NewSelect().Model(&entry).
		Relation("Postings", func(q *bun.SelectQuery) *bun.SelectQuery { return q.Order("line_number") }).
		Relation("PositionMovements").
		Where("je.id = ?", id).Scan(s.T().Context())
	require.NoError(s.T(), err)
	return entry
}

func (s *UnitTestSuite) close(account any) {
	s.T().Helper()

	_, err := s.DB.NewUpdate().Model(account).Set("closed_at = now()").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)
}

// Test_ReplaceJournalEntry checks what a replace writes and what it keeps.
// Every case uses an asset of its own, so no case depends on what another left
// behind.
func (s *UnitTestSuite) Test_ReplaceJournalEntry() {
	h := s.newHeld()
	counterparty := &model.Counterparty{Name: "Broker"}
	temporaltesting.Insert(s.T(), s.DB, counterparty)

	tests := []struct {
		name string
		// oldUnits and newUnits are the units the entry moves before and after
		// the replace, and zero means it has no movement.
		oldUnits, newUnits int64
		// soldSince is the units a later entry sells, and zero means there is none.
		soldSince int64
		// wantNegative is the units the warning reports, and zero means no warning.
		wantNegative int64
	}{
		{name: "an entry without movements"},
		{name: "a trade with other units", oldUnits: 10, newUnits: 12},
		{name: "a trade becomes a plain entry", oldUnits: 10},
		{name: "a plain entry becomes a trade", newUnits: 5},
		{name: "a buy lowered below what was sold since", oldUnits: 10, newUnits: 4, soldSince: -10, wantNegative: -6},
		{name: "a buy removed after its units were sold", oldUnits: 10, soldSince: -10, wantNegative: -10},
		{name: "a buy raised with a later sale", oldUnits: 10, newUnits: 20, soldSince: -10},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			asset := s.newAsset(tc.name)
			createdAt := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
			entry := &model.JournalEntry{
				OccurredAt: deleteDay, CreatedAt: createdAt, Description: ptr.NewPtr("as recorded"), Metadata: map[string]string{"source": "recorded"},
				IdempotencyKey: ptr.NewPtr("replace-" + tc.name), ExternalID: ptr.NewPtr("broker-" + tc.name), CounterpartyID: &counterparty.ID,
			}
			var oldMovements []*model.PositionMovement
			if tc.oldUnits != 0 {
				oldMovements = append(oldMovements, &model.PositionMovement{FinancialAccountID: h.account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(tc.oldUnits)})
			}
			temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry, []model.Posting{
				{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
				{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
			}, oldMovements)
			if tc.soldSince != 0 {
				s.seedEntry(h, asset, model.JournalEntryPosted, tc.soldSince, replaceDay.AddDate(0, 0, 1))
			}

			activityResult, err := s.Env.ExecuteActivity(s.ja.ReplaceJournalEntry, replaceInput(h, entry.ID, asset, 25, tc.newUnits))

			require.NoError(s.T(), err)
			var result lt.ReplaceJournalEntryResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Equal(s.T(), entry.ID, result.JournalEntryID)

			stored := s.storedEntry(entry.ID)
			require.Equal(s.T(), model.JournalEntryPosted, stored.Status)
			require.True(s.T(), replaceDay.Equal(stored.OccurredAt))
			require.Equal(s.T(), "as corrected", *stored.Description)
			require.Empty(s.T(), stored.Metadata)
			require.True(s.T(), createdAt.Equal(stored.CreatedAt))
			require.True(s.T(), entry.PostedAt.Equal(*stored.PostedAt))
			require.Equal(s.T(), *entry.IdempotencyKey, *stored.IdempotencyKey)
			require.Equal(s.T(), *entry.ExternalID, *stored.ExternalID)
			require.Equal(s.T(), counterparty.ID, *stored.CounterpartyID)

			require.Len(s.T(), stored.Postings, 2)
			for _, posting := range stored.Postings {
				require.True(s.T(), decimal.NewFromInt(25).Equal(posting.Amount), "posting of %s", posting.Amount)
			}

			if tc.newUnits == 0 {
				require.Empty(s.T(), stored.PositionMovements)
				require.Empty(s.T(), result.PositionMovementIDs)
			} else {
				require.Len(s.T(), stored.PositionMovements, 1)
				movement := stored.PositionMovements[0]
				require.Equal(s.T(), []int64{movement.ID}, result.PositionMovementIDs)
				require.True(s.T(), decimal.NewFromInt(tc.newUnits).Equal(movement.UnitsDelta))
				require.True(s.T(), replaceDay.Equal(movement.OccurredAt))
				require.True(s.T(), createdAt.Equal(movement.CreatedAt), "a movement keeps the entry's place, got %s", movement.CreatedAt)
				require.Equal(s.T(), *entry.ExternalID, *movement.ExternalID)
			}

			if tc.wantNegative == 0 {
				require.Empty(s.T(), result.Warnings)
				return
			}
			require.Len(s.T(), result.Warnings, 1)
			warning := result.Warnings[0]
			require.Equal(s.T(), h.account.ID, warning.FinancialAccountID)
			require.Equal(s.T(), asset.ID, warning.AssetID)
			require.True(s.T(), warning.Units.Equal(decimal.NewFromInt(tc.wantNegative)), "holds %s units", warning.Units)
		})
	}
}

// An entry may stay on a closed account it already touches, and may not move
// onto one.
func (s *UnitTestSuite) Test_ReplaceJournalEntry_ClosedAccounts() {
	h := s.newHeld()

	tests := []struct {
		name string
		// closeTouched closes the accounts of the entry before the replace.
		closeTouched bool
		// toClosedLedger and toClosedFinancial move the entry onto a closed
		// account it does not touch.
		toClosedLedger, toClosedFinancial bool
		wantErr                           string
	}{
		{name: "stays on its closed accounts", closeTouched: true},
		{name: "moves onto a closed ledger account", toClosedLedger: true, wantErr: "is closed"},
		{name: "moves onto a closed financial account", toClosedFinancial: true, wantErr: "is closed"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			asset := s.newAsset(tc.name)
			// The case has accounts of its own, since closing them is final.
			mine := held{
				account: &model.FinancialAccount{InstitutionID: h.account.InstitutionID, Name: tc.name},
				debit:   testLedgerAccount(s.T(), s.DB, h.debit.CurrencyID, tc.name+" debit", model.LedgerAccountAsset),
				credit:  h.credit,
			}
			temporaltesting.Insert(s.T(), s.DB, mine.account)
			seeded := s.seedEntry(mine, asset, model.JournalEntryPosted, 10, deleteDay)

			target := mine
			if tc.closeTouched {
				s.close(mine.account)
				s.close(mine.debit)
			}
			if tc.toClosedLedger {
				target.debit = testLedgerAccount(s.T(), s.DB, h.debit.CurrencyID, tc.name+" closed", model.LedgerAccountAsset)
				s.close(target.debit)
			}
			if tc.toClosedFinancial {
				target.account = &model.FinancialAccount{InstitutionID: h.account.InstitutionID, Name: tc.name + " closed"}
				temporaltesting.Insert(s.T(), s.DB, target.account)
				s.close(target.account)
			}

			_, err := s.Env.ExecuteActivity(s.ja.ReplaceJournalEntry, replaceInput(target, seeded.entry.ID, asset, 25, 12))

			stored := s.storedEntry(seeded.entry.ID)
			if tc.wantErr != "" {
				require.ErrorContains(s.T(), err, tc.wantErr)
				require.True(s.T(), deleteDay.Equal(stored.OccurredAt), "the entry is left as it was")
				require.True(s.T(), decimal.NewFromInt(10).Equal(stored.Postings[0].Amount))
				require.Equal(s.T(), seeded.movement.ID, stored.PositionMovements[0].ID)
				return
			}
			require.NoError(s.T(), err)
			require.True(s.T(), decimal.NewFromInt(25).Equal(stored.Postings[0].Amount))
			require.True(s.T(), decimal.NewFromInt(12).Equal(stored.PositionMovements[0].UnitsDelta))
		})
	}
}

func (s *UnitTestSuite) Test_ReplaceJournalEntry_Rejected() {
	h := s.newHeld()
	posted := s.seedEntry(h, h.asset, model.JournalEntryPosted, 10, deleteDay)
	draft := s.seedEntry(h, h.asset, model.JournalEntryDraft, 0, deleteDay)

	valid := replaceInput(h, posted.entry.ID, h.asset, 25, 12)
	missing, ofDraft, unbalanced, noPostings, unknownAccount, unknownAsset, unknownFinancialAccount := valid, valid, valid, valid, valid, valid, valid
	missing.JournalEntryID = 987654
	ofDraft.JournalEntryID = draft.entry.ID
	unbalanced.Postings = []lt.PostingInput{valid.Postings[0], {LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(24), Side: model.PostingCredit}}
	noPostings.Postings = nil
	unknownAccount.Postings = []lt.PostingInput{valid.Postings[0], {LedgerAccountID: 987654, Amount: decimal.NewFromInt(25), Side: model.PostingCredit}}
	unknownAsset.Movements = []lt.ReplaceMovementInput{{Kind: model.PositionMovementTrade, FinancialAccountID: h.account.ID, AssetID: 987654, UnitsDelta: decimal.NewFromInt(1)}}
	unknownFinancialAccount.Movements = []lt.ReplaceMovementInput{{Kind: model.PositionMovementTrade, FinancialAccountID: 987654, AssetID: h.asset.ID, UnitsDelta: decimal.NewFromInt(1)}}

	tests := []struct {
		name    string
		input   lt.ReplaceJournalEntryInput
		wantErr string
	}{
		{name: "missing entry", input: missing, wantErr: "JournalEntry not found"},
		{name: "draft entry", input: ofDraft, wantErr: "is not posted"},
		{name: "unbalanced postings", input: unbalanced, wantErr: "do not balance"},
		{name: "no postings", input: noPostings, wantErr: "at least two postings"},
		{name: "unknown ledger account", input: unknownAccount, wantErr: "LedgerAccount not found"},
		{name: "unknown asset", input: unknownAsset, wantErr: "Asset"},
		{name: "unknown financial account", input: unknownFinancialAccount, wantErr: "FinancialAccount not found"},
	}

	for _, tc := range tests {
		s.Run(tc.name, func() {
			_, err := s.Env.ExecuteActivity(s.ja.ReplaceJournalEntry, tc.input)

			require.ErrorContains(s.T(), err, tc.wantErr)
			s.requireEntryKept(posted)
			stored := s.storedEntry(posted.entry.ID)
			require.True(s.T(), decimal.NewFromInt(10).Equal(stored.Postings[0].Amount), "the entry is left as it was")
			require.Equal(s.T(), posted.movement.ID, stored.PositionMovements[0].ID)
		})
	}
}

// Test_ReplaceJournalEntry_Idempotent checks that a replace run again leaves
// the same entry and reports the same warnings.
func (s *UnitTestSuite) Test_ReplaceJournalEntry_Idempotent() {
	h := s.newHeld()
	purchase := s.seedEntry(h, h.asset, model.JournalEntryPosted, 10, deleteDay)
	s.seedEntry(h, h.asset, model.JournalEntryPosted, -10, replaceDay.AddDate(0, 0, 1))
	input := replaceInput(h, purchase.entry.ID, h.asset, 25, 4)

	first, err := s.Env.ExecuteActivity(s.ja.ReplaceJournalEntry, input)
	require.NoError(s.T(), err)
	var created lt.ReplaceJournalEntryResult
	require.NoError(s.T(), first.Get(&created))
	afterFirst := s.storedEntry(purchase.entry.ID)

	second, err := s.Env.ExecuteActivity(s.ja.ReplaceJournalEntry, input)
	require.NoError(s.T(), err)
	var retried lt.ReplaceJournalEntryResult
	require.NoError(s.T(), second.Get(&retried))
	afterSecond := s.storedEntry(purchase.entry.ID)

	require.Len(s.T(), created.Warnings, 1)
	require.Equal(s.T(), created.Warnings, retried.Warnings)
	require.Equal(s.T(), created.JournalEntryID, retried.JournalEntryID)
	require.Len(s.T(), retried.PositionMovementIDs, 1)

	require.True(s.T(), afterFirst.OccurredAt.Equal(afterSecond.OccurredAt))
	require.True(s.T(), afterFirst.CreatedAt.Equal(afterSecond.CreatedAt))
	require.Len(s.T(), afterSecond.Postings, 2)
	require.True(s.T(), decimal.NewFromInt(25).Equal(afterSecond.Postings[0].Amount))
	require.Len(s.T(), afterSecond.PositionMovements, 1)
	require.True(s.T(), decimal.NewFromInt(4).Equal(afterSecond.PositionMovements[0].UnitsDelta))
	require.True(s.T(), afterFirst.PositionMovements[0].CreatedAt.Equal(afterSecond.PositionMovements[0].CreatedAt))
	require.Equal(s.T(), 2, s.Count[model.JournalEntry]())
	require.Equal(s.T(), 4, s.Count[model.Posting]())
	require.Equal(s.T(), 2, s.Count[model.PositionMovement]())
}
