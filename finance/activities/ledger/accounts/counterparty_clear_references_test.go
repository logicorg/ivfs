package accounts

import (
	"slices"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// fixture records journal entries that name counterparties, on two
// accounts inserted once per test.
type fixture struct {
	s               *UnitTestSuite
	cash, groceries *model.LedgerAccount
}

func newFixture(s *UnitTestSuite) *fixture {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	e := &fixture{
		s:         s,
		cash:      &model.LedgerAccount{CurrencyID: eur.ID, Name: "Cash", Kind: model.LedgerAccountAsset},
		groceries: &model.LedgerAccount{CurrencyID: eur.ID, Name: "Groceries", Kind: model.LedgerAccountExpense},
	}
	temporaltesting.Insert(s.T(), s.DB, e.cash)
	temporaltesting.Insert(s.T(), s.DB, e.groceries)
	return e
}

// post records an expense of 10 paid to the counterparty.
func (e *fixture) post(counterpartyID int64) *model.JournalEntry {
	entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), CounterpartyID: &counterpartyID}
	temporaltesting.InsertPostedJournalEntry(e.s.T(), e.s.DB, entry,
		model.Posting{LedgerAccountID: e.groceries.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: e.cash.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	)
	return entry
}

// reverse records the entry that reverses entry, naming the same counterparty
// as the reverse activity does.
func (e *fixture) reverse(entry *model.JournalEntry) *model.JournalEntry {
	reversal := &model.JournalEntry{OccurredAt: entry.OccurredAt, CounterpartyID: entry.CounterpartyID, ReversalOfEntryID: &entry.ID}
	temporaltesting.InsertPostedJournalEntry(e.s.T(), e.s.DB, reversal,
		model.Posting{LedgerAccountID: e.cash.ID, Amount: decimal.NewFromInt(10), Side: model.PostingDebit},
		model.Posting{LedgerAccountID: e.groceries.ID, Amount: decimal.NewFromInt(10), Side: model.PostingCredit},
	)
	return reversal
}

// draft records a draft entry, without postings, naming the counterparty.
func (e *fixture) draft(counterpartyID int64) *model.JournalEntry {
	entry := &model.JournalEntry{OccurredAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC), CounterpartyID: &counterpartyID, Status: model.JournalEntryDraft}
	temporaltesting.Insert(e.s.T(), e.s.DB, entry)
	return entry
}

// deleteAll deletes entries, latest first, so that a reversal goes before the
// entry it reverses.
func (e *fixture) deleteAll(entries []*model.JournalEntry) {
	for _, entry := range slices.Backward(entries) {
		temporaltesting.DeleteJournalEntry(e.s.T(), e.s.DB, entry.ID)
	}
}

func (s *UnitTestSuite) Test_ClearCounterpartyReferences() {
	fixture := newFixture(s)

	tests := map[string]struct {
		// posted, reversed, and drafts are the entries naming the counterparty:
		// reversed counts posted entries that are reversed, each with its reversal.
		posted, reversed, drafts int
		// others are the posted entries naming another counterparty, which keep it.
		others     int
		expCleared int64
	}{
		"posted entries":                    {posted: 2, expCleared: 2},
		"a draft":                           {drafts: 1, expCleared: 1},
		"a reversed entry and its reversal": {reversed: 1, expCleared: 2},
		"only the entries of the counterparty": {
			posted: 1, drafts: 1, others: 2, expCleared: 2,
		},
		"no entries": {others: 1, expCleared: 0},
	}

	for name, tc := range tests {
		payee := &model.Counterparty{Name: "Tesco"}
		other := &model.Counterparty{Name: "Lidl"}
		temporaltesting.Insert(s.T(), s.DB, payee)
		temporaltesting.Insert(s.T(), s.DB, other)
		var deletingCounterpartyEntries, otherCounterpartiesEntries []*model.JournalEntry
		for range tc.posted {
			deletingCounterpartyEntries = append(deletingCounterpartyEntries, fixture.post(payee.ID))
		}
		for range tc.reversed {
			entry := fixture.post(payee.ID)
			deletingCounterpartyEntries = append(deletingCounterpartyEntries, entry, fixture.reverse(entry))
		}
		for range tc.drafts {
			deletingCounterpartyEntries = append(deletingCounterpartyEntries, fixture.draft(payee.ID))
		}
		for range tc.others {
			otherCounterpartiesEntries = append(otherCounterpartiesEntries, fixture.post(other.ID))
		}

		s.Run(name, func() {
			val, err := s.Env.ExecuteActivity(s.aa.ClearCounterpartyReferences, lt.ClearCounterpartyReferencesInput{CounterpartyID: payee.ID})
			require.NoError(s.T(), err)
			var result lt.ClearCounterpartyReferencesResult
			require.NoError(s.T(), val.Get(&result))
			require.Equal(s.T(), tc.expCleared, result.Cleared)

			for _, entry := range deletingCounterpartyEntries {
				var stored model.JournalEntry
				temporaltesting.Retrieve(s.T(), s.DB, &stored, entry.ID)
				require.Nil(s.T(), stored.CounterpartyID, "entry %d still names the counterparty", entry.ID)
				require.Equal(s.T(), entry.Status, stored.Status)
				if entry.Status == model.JournalEntryPosted {
					postings := s.CountWhere[model.Posting]("journal_entry_id = ?", entry.ID)
					require.Equal(s.T(), 2, postings, "entry %d lost postings", entry.ID)
				}
			}
			for _, entry := range otherCounterpartiesEntries {
				var stored model.JournalEntry
				temporaltesting.Retrieve(s.T(), s.DB, &stored, entry.ID)
				require.Equal(s.T(), &other.ID, stored.CounterpartyID, "entry %d of another counterparty lost it", entry.ID)
			}
		})

		fixture.deleteAll(append(deletingCounterpartyEntries, otherCounterpartiesEntries...))
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Counterparty)(nil), payee.ID)
		temporaltesting.DeleteIgnoreNotFound(s.T(), s.DB, (*model.Counterparty)(nil), other.ID)
	}
}

func (s *UnitTestSuite) Test_ClearCounterpartyReferences_InvalidInput() {
	_, err := s.Env.ExecuteActivity(s.aa.ClearCounterpartyReferences, lt.ClearCounterpartyReferencesInput{})
	require.ErrorContains(s.T(), err, "validation error")
}

// Test_ClearCounterpartyReferences_Idempotency checks that a retry finds
// nothing left to clear, reports it, and leaves the entries as the first run
// left them.
func (s *UnitTestSuite) Test_ClearCounterpartyReferences_Idempotency() {
	fixture := newFixture(s)
	payee := &model.Counterparty{Name: "Tesco"}
	temporaltesting.Insert(s.T(), s.DB, payee)
	entries := []*model.JournalEntry{fixture.post(payee.ID), fixture.draft(payee.ID)}
	input := lt.ClearCounterpartyReferencesInput{CounterpartyID: payee.ID}

	for _, expCleared := range []int64{2, 0} {
		val, err := s.Env.ExecuteActivity(s.aa.ClearCounterpartyReferences, input)
		require.NoError(s.T(), err)
		var result lt.ClearCounterpartyReferencesResult
		require.NoError(s.T(), val.Get(&result))
		require.Equal(s.T(), expCleared, result.Cleared)
	}

	for _, entry := range entries {
		var stored model.JournalEntry
		temporaltesting.Retrieve(s.T(), s.DB, &stored, entry.ID)
		require.Nil(s.T(), stored.CounterpartyID)
	}
}
