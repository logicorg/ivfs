package journal

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

var deleteDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// seededEntry is an entry a case has inserted, with the movement it carries.
type seededEntry struct {
	entry    *model.JournalEntry
	movement *model.PositionMovement
}

// seedEntry inserts an entry of ten euro between the ledger accounts of h, with
// a movement of units of the asset when units is not zero. A reversal inverts
// the sides and the units of what it reverses and refers to both.
func (s *UnitTestSuite) seedEntry(h held, asset *model.Asset, status model.JournalEntryStatus, units int64, at time.Time, reverses *seededEntry) seededEntry {
	s.T().Helper()

	entry := &model.JournalEntry{OccurredAt: at}
	debit, credit := model.PostingDebit, model.PostingCredit
	if reverses != nil {
		entry.ReversalOfEntryID = &reverses.entry.ID
		debit, credit = credit, debit
		units = -units
	}
	postings := []model.Posting{
		{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(10), Side: debit},
		{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(10), Side: credit},
	}
	var movement *model.PositionMovement
	var movements []*model.PositionMovement
	if units != 0 {
		movement = &model.PositionMovement{FinancialAccountID: h.account.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(units)}
		if reverses != nil {
			movement.ReversalOfMovementID = &reverses.movement.ID
		}
		movements = append(movements, movement)
	}

	if status == model.JournalEntryDraft {
		entry.Status = model.JournalEntryDraft
		temporaltesting.Insert(s.T(), s.DB, entry)
		for n := range postings {
			postings[n].JournalEntryID, postings[n].LineNumber = entry.ID, n+1
			temporaltesting.Insert(s.T(), s.DB, &postings[n])
		}
		return seededEntry{entry: entry}
	}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, entry, postings, movements)
	return seededEntry{entry: entry, movement: movement}
}

func (s *UnitTestSuite) newAsset(name string) *model.Asset {
	asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	return asset
}

func (s *UnitTestSuite) rowsOfEntry(entryID int64) (entries, postings, movements int64) {
	s.T().Helper()

	var err error
	entries, err = s.DB.NewSelect().Model((*model.JournalEntry)(nil)).Where("id = ?", entryID).Count(s.T().Context())
	require.NoError(s.T(), err)
	postings, err = s.DB.NewSelect().Model((*model.Posting)(nil)).Where("journal_entry_id = ?", entryID).Count(s.T().Context())
	require.NoError(s.T(), err)
	movements, err = s.DB.NewSelect().Model((*model.PositionMovement)(nil)).Where("journal_entry_id = ?", entryID).Count(s.T().Context())
	require.NoError(s.T(), err)
	return entries, postings, movements
}

func (s *UnitTestSuite) requireEntryGone(entry seededEntry) {
	s.T().Helper()

	entries, postings, movements := s.rowsOfEntry(entry.entry.ID)
	require.Zero(s.T(), entries+postings+movements, "entry %d left rows behind", entry.entry.ID)
}

func (s *UnitTestSuite) requireEntryKept(entry seededEntry) {
	s.T().Helper()

	entries, postings, movements := s.rowsOfEntry(entry.entry.ID)
	require.Equal(s.T(), 1, entries)
	require.Equal(s.T(), 2, postings)
	if entry.movement != nil {
		require.Equal(s.T(), 1, movements)
	}
}

// Test_DeleteJournalEntry checks that an entry of any status is deleted with
// its postings and movements, that the reversals that follow it go with it, and
// that what is left is untouched. Every case uses an asset of its own, so no
// case depends on what another left behind.
func (s *UnitTestSuite) Test_DeleteJournalEntry() {
	h := s.newHeld()

	tests := map[string]struct {
		// seed inserts the entries of the case, and returns the one to delete, the
		// ones that go with it, and the ones that stay.
		seed func(asset *model.Asset) (target seededEntry, gone, kept []seededEntry)
		// wantUnits is the units a warning reports, and none means no warning.
		wantUnits *int64
	}{
		"draft entry": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			draft := s.seedEntry(h, asset, model.JournalEntryDraft, 0, deleteDay, nil)
			return draft, []seededEntry{draft}, nil
		}},
		"posted entry": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			posted := s.seedEntry(h, asset, model.JournalEntryPosted, 0, deleteDay, nil)
			return posted, []seededEntry{posted}, nil
		}},
		"posted entry with a movement": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			trade := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
			return trade, []seededEntry{trade}, nil
		}},
		"another entry stays": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			target := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
			other := s.seedEntry(h, asset, model.JournalEntryPosted, 5, deleteDay.AddDate(0, 0, 1), nil)
			return target, []seededEntry{target}, []seededEntry{other}
		}},
		"entry with a reversal chain": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			original := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
			reversal := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay.AddDate(0, 0, 1), &original)
			again := s.seedEntry(h, asset, model.JournalEntryPosted, -10, deleteDay.AddDate(0, 0, 2), &reversal)
			return original, []seededEntry{original, reversal, again}, nil
		}},
		"reversal alone": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			original := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
			reversal := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay.AddDate(0, 0, 1), &original)
			return reversal, []seededEntry{reversal}, []seededEntry{original}
		}},
		"middle of a reversal chain": {seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			original := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
			reversal := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay.AddDate(0, 0, 1), &original)
			again := s.seedEntry(h, asset, model.JournalEntryPosted, -10, deleteDay.AddDate(0, 0, 2), &reversal)
			return reversal, []seededEntry{reversal, again}, []seededEntry{original}
		}},
		"units sold since": {
			seed: func(asset *model.Asset) (seededEntry, []seededEntry, []seededEntry) {
				purchase := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)
				sale := s.seedEntry(h, asset, model.JournalEntryPosted, -10, deleteDay.AddDate(0, 0, 1), nil)
				return purchase, []seededEntry{purchase}, []seededEntry{sale}
			},
			wantUnits: ptr.Int64(-10),
		},
		"missing entry": {seed: func(*model.Asset) (seededEntry, []seededEntry, []seededEntry) {
			return seededEntry{entry: &model.JournalEntry{ID: 987654}}, nil, nil
		}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := s.newAsset(name)
			target, gone, kept := tc.seed(asset)

			activityResult, err := s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, lt.DeleteJournalEntryInput{JournalEntryID: target.entry.ID})

			require.NoError(s.T(), err)
			var result lt.DeleteJournalEntryResult
			require.NoError(s.T(), activityResult.Get(&result))
			for _, entry := range gone {
				s.requireEntryGone(entry)
			}
			for _, entry := range kept {
				s.requireEntryKept(entry)
			}
			if tc.wantUnits == nil {
				require.Empty(s.T(), result.Warnings)
				return
			}
			require.Len(s.T(), result.Warnings, 1)
			warning := result.Warnings[0]
			require.Equal(s.T(), h.account.ID, warning.FinancialAccountID)
			require.Equal(s.T(), asset.ID, warning.AssetID)
			require.Equal(s.T(), asset.Name, warning.AssetName)
			require.True(s.T(), warning.Units.Equal(decimal.NewFromInt(*tc.wantUnits)), "holds %s units", warning.Units)
		})
	}
}

func (s *UnitTestSuite) Test_DeleteJournalEntry_Rejected() {
	tests := map[string]lt.DeleteJournalEntryInput{
		"no entry":       {},
		"negative entry": {JournalEntryID: -1},
	}

	for name, input := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, input)

			require.Error(s.T(), err)
		})
	}
}

// Closing is final, so an entry made while its accounts were open has to remain
// deletable, as it remains reversible.
func (s *UnitTestSuite) Test_DeleteJournalEntry_ClosedAccounts() {
	h := s.newHeld()
	asset := h.asset
	purchase := s.seedEntry(h, asset, model.JournalEntryPosted, 10, deleteDay, nil)

	closedAt := time.Now().UTC()
	h.account.ClosedAt, h.debit.ClosedAt = &closedAt, &closedAt
	_, err := s.DB.NewUpdate().Model(h.account).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)
	_, err = s.DB.NewUpdate().Model(h.debit).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, lt.DeleteJournalEntryInput{JournalEntryID: purchase.entry.ID})

	require.NoError(s.T(), err)
	s.requireEntryGone(purchase)
}

// Test_DeleteJournalEntry_Idempotent checks that deleting again returns the
// same warnings and changes nothing, even after the state around it changed.
func (s *UnitTestSuite) Test_DeleteJournalEntry_Idempotent() {
	h := s.newHeld()
	purchase := s.seedEntry(h, h.asset, model.JournalEntryPosted, 10, deleteDay, nil)
	sale := s.seedEntry(h, h.asset, model.JournalEntryPosted, -10, deleteDay.AddDate(0, 0, 1), nil)
	input := lt.DeleteJournalEntryInput{JournalEntryID: purchase.entry.ID}

	first, err := s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, input)
	require.NoError(s.T(), err)
	var created lt.DeleteJournalEntryResult
	require.NoError(s.T(), first.Get(&created))
	require.Len(s.T(), created.Warnings, 1)

	second, err := s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, input)
	require.NoError(s.T(), err)
	var retried lt.DeleteJournalEntryResult
	require.NoError(s.T(), second.Get(&retried))

	require.Empty(s.T(), retried.Warnings, "a delete that finds nothing touches no position")
	s.requireEntryGone(purchase)
	s.requireEntryKept(sale)
	require.Equal(s.T(), 1, s.Count[model.JournalEntry]())
}

// Deleting a reversal puts the effect of the entry it reversed back, so that
// entry can be reversed again.
func (s *UnitTestSuite) Test_DeleteJournalEntry_ReversalAloneAllowsReversingAgain() {
	h := s.newHeld()
	original := s.seedEntry(h, h.asset, model.JournalEntryPosted, 10, deleteDay, nil)
	reversal := s.seedEntry(h, h.asset, model.JournalEntryPosted, 10, deleteDay.AddDate(0, 0, 1), &original)

	_, err := s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: original.entry.ID, IdempotencyKey: "again-before"})
	require.Error(s.T(), err, "the original is reversed while its reversal exists")

	_, err = s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, lt.DeleteJournalEntryInput{JournalEntryID: reversal.entry.ID})
	require.NoError(s.T(), err)

	_, err = s.Env.ExecuteActivity(s.ja.ReverseJournalEntry, lt.ReverseJournalEntryInput{JournalEntryID: original.entry.ID, IdempotencyKey: "again-after"})
	require.NoError(s.T(), err)
}

// Deleting an entry frees its idempotency key, so the operation can be recorded
// anew, which is how a wrong opening balance is replaced (ADR 0011).
func (s *UnitTestSuite) Test_DeleteJournalEntry_FreesIdempotencyKey() {
	h := s.newHeld()
	const key = "opening-balance:1"
	posted := &model.JournalEntry{OccurredAt: deleteDay, IdempotencyKey: ptr.NewPtr(key)}
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, posted,
		model.Posting{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	)

	_, err := s.Env.ExecuteActivity(s.ja.DeleteJournalEntry, lt.DeleteJournalEntryInput{JournalEntryID: posted.ID})
	require.NoError(s.T(), err)

	recorded, err := s.Env.ExecuteActivity(s.ja.PostJournalEntry, lt.PostJournalEntryInput{
		IdempotencyKey: key, OccurredAt: deleteDay,
		Postings: []lt.PostingInput{
			{LedgerAccountID: h.debit.ID, Amount: decimal.NewFromInt(20), Side: model.PostingDebit},
			{LedgerAccountID: h.credit.ID, Amount: decimal.NewFromInt(20), Side: model.PostingCredit},
		},
	})
	require.NoError(s.T(), err)
	var result lt.PostJournalEntryResult
	require.NoError(s.T(), recorded.Get(&result))
	require.NotEqual(s.T(), posted.ID, result.JournalEntryID)
	var postings []model.Posting
	require.NoError(s.T(), s.DB.NewSelect().Model(&postings).Where("journal_entry_id = ?", result.JournalEntryID).Scan(s.T().Context()))
	require.True(s.T(), postings[0].Amount.Equal(decimal.NewFromInt(20)), "the entry is recorded anew, not returned")
}
