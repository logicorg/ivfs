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

var listDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

func listAt(day int) time.Time { return listDay.AddDate(0, 0, day) }

// journalFixture is the ledger the list tests read. It is only read, so the
// cases of a test share one.
type journalFixture struct {
	cash, groceries, salary, rent, tax *model.LedgerAccount
	tesco, employer                    *model.Counterparty
	// entries are in the order the list returns them, oldest first:
	//
	//	0  day 1  posted  salary: cash 80 and tax 20 against salary 100  (employer, payroll-1)
	//	1  day 2  posted  groceries 20 against cash                      (tesco, tesco-1)
	//	2  day 3  posted  groceries 30 against cash                      (tesco, tesco-2)
	//	3  day 3  posted  rent 50 against cash                           (same instant as 2)
	//	4  day 4  draft   groceries 5 against cash
	//	5  day 5  draft   rent 7 against cash
	entries []*model.JournalEntry
}

func (s *UnitTestSuite) seedJournal() journalFixture {
	s.T().Helper()

	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	f := journalFixture{
		cash:      testLedgerAccount(s.T(), s.DB, eur.ID, "cash", model.LedgerAccountAsset),
		groceries: testLedgerAccount(s.T(), s.DB, eur.ID, "groceries", model.LedgerAccountExpense),
		salary:    testLedgerAccount(s.T(), s.DB, eur.ID, "salary", model.LedgerAccountIncome),
		rent:      testLedgerAccount(s.T(), s.DB, eur.ID, "rent", model.LedgerAccountExpense),
		tax:       testLedgerAccount(s.T(), s.DB, eur.ID, "tax", model.LedgerAccountExpense),
		tesco:     &model.Counterparty{Name: "Tesco"},
		employer:  &model.Counterparty{Name: "Employer"},
	}
	temporaltesting.Insert(s.T(), s.DB, f.tesco)
	temporaltesting.Insert(s.T(), s.DB, f.employer)

	debit := func(account *model.LedgerAccount, amount int64) model.Posting {
		return model.Posting{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingDebit}
	}
	credit := func(account *model.LedgerAccount, amount int64) model.Posting {
		return model.Posting{LedgerAccountID: account.ID, Amount: decimal.NewFromInt(amount), Side: model.PostingCredit}
	}
	entry := func(day int, counterparty *model.Counterparty, externalID string) *model.JournalEntry {
		e := &model.JournalEntry{OccurredAt: listAt(day)}
		if counterparty != nil {
			e.CounterpartyID = &counterparty.ID
		}
		if externalID != "" {
			e.ExternalID = ptr.NewPtr(externalID)
		}
		return e
	}

	f.entries = []*model.JournalEntry{
		entry(1, f.employer, "payroll-1"),
		entry(2, f.tesco, "tesco-1"),
		entry(3, f.tesco, "tesco-2"),
		entry(3, nil, ""),
		entry(4, nil, ""),
		entry(5, nil, ""),
	}
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, f.entries[0], debit(f.cash, 80), debit(f.tax, 20), credit(f.salary, 100))
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, f.entries[1], debit(f.groceries, 20), credit(f.cash, 20))
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, f.entries[2], debit(f.groceries, 30), credit(f.cash, 30))
	temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, f.entries[3], debit(f.rent, 50), credit(f.cash, 50))
	s.insertDraft(f.entries[4], debit(f.groceries, 5), credit(f.cash, 5))
	s.insertDraft(f.entries[5], debit(f.rent, 7), credit(f.cash, 7))
	return f
}

// insertDraft inserts an entry with its postings, left as a draft.
func (s *UnitTestSuite) insertDraft(entry *model.JournalEntry, postings ...model.Posting) {
	s.T().Helper()

	entry.Status = model.JournalEntryDraft
	temporaltesting.Insert(s.T(), s.DB, entry)
	for n := range postings {
		postings[n].JournalEntryID, postings[n].LineNumber = entry.ID, n+1
	}
	_, err := s.DB.NewInsert().Model(&postings).Exec(s.T().Context())
	require.NoError(s.T(), err)
}

func (s *UnitTestSuite) listEntries(params lt.ListJournalEntriesParams) lt.ListJournalEntriesResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.ja.ListJournalEntries, params)
	require.NoError(s.T(), err)
	var result lt.ListJournalEntriesResult
	require.NoError(s.T(), activityResult.Get(&result))
	return result
}

func entryIDs(entries []model.JournalEntry) []int64 {
	ids := make([]int64, len(entries))
	for n := range entries {
		ids[n] = entries[n].ID
	}
	return ids
}

// idsOf returns the ids of the entries at the given positions of the fixture.
func (f journalFixture) idsOf(positions ...int) []int64 {
	ids := make([]int64, len(positions))
	for n, position := range positions {
		ids[n] = f.entries[position].ID
	}
	return ids
}

// Test_ListJournalEntries checks each filter of the list, both directions, and
// that the filters combine.
func (s *UnitTestSuite) Test_ListJournalEntries() {
	f := s.seedJournal()

	tests := map[string]struct {
		params lt.ListJournalEntriesParams
		// want are the positions in the fixture of the entries listed, in order.
		want []int
	}{
		"everything, oldest first": {
			params: lt.ListJournalEntriesParams{},
			want:   []int{0, 1, 2, 3, 4, 5},
		},
		"everything, newest first": {
			params: lt.ListJournalEntriesParams{Descending: true},
			want:   []int{5, 4, 3, 2, 1, 0},
		},
		"a ledger account with several entries": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.groceries.ID},
			want:   []int{1, 2, 4},
		},
		"a ledger account that every entry but one has": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.cash.ID},
			want:   []int{0, 1, 2, 3, 4, 5},
		},
		"a ledger account of one posting in one entry": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.tax.ID},
			want:   []int{0},
		},
		"a counterparty": {
			params: lt.ListJournalEntriesParams{CounterpartyID: f.tesco.ID},
			want:   []int{1, 2},
		},
		"an external id": {
			params: lt.ListJournalEntriesParams{ExternalID: "tesco-2"},
			want:   []int{2},
		},
		"an external id that only starts the same": {
			params: lt.ListJournalEntriesParams{ExternalID: "tesco"},
			want:   []int{},
		},
		"posted entries": {
			params: lt.ListJournalEntriesParams{Statuses: []model.JournalEntryStatus{model.JournalEntryPosted}},
			want:   []int{0, 1, 2, 3},
		},
		"draft entries": {
			params: lt.ListJournalEntriesParams{Statuses: []model.JournalEntryStatus{model.JournalEntryDraft}},
			want:   []int{4, 5},
		},
		"several statuses": {
			params: lt.ListJournalEntriesParams{Statuses: []model.JournalEntryStatus{model.JournalEntryDraft, model.JournalEntryPosted}},
			want:   []int{0, 1, 2, 3, 4, 5},
		},
		"a range includes its start": {
			params: lt.ListJournalEntriesParams{From: listAt(3)},
			want:   []int{2, 3, 4, 5},
		},
		"a range excludes its end": {
			params: lt.ListJournalEntriesParams{To: listAt(3)},
			want:   []int{0, 1},
		},
		"a range between two dates": {
			params: lt.ListJournalEntriesParams{From: listAt(2), To: listAt(4)},
			want:   []int{1, 2, 3},
		},
		"a range after the entries": {
			params: lt.ListJournalEntriesParams{From: listAt(10)},
			want:   []int{},
		},
		"an account and a status together": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.groceries.ID, Statuses: []model.JournalEntryStatus{model.JournalEntryPosted}},
			want:   []int{1, 2},
		},
		"an account, a counterparty and a range together": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.cash.ID, CounterpartyID: f.tesco.ID, From: listAt(2), To: listAt(3)},
			want:   []int{1},
		},
		"filters that no entry satisfies together": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.salary.ID, CounterpartyID: f.tesco.ID},
			want:   []int{},
		},
		"a filtered list, newest first": {
			params: lt.ListJournalEntriesParams{LedgerAccountID: f.rent.ID, Descending: true},
			want:   []int{5, 3},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			result := s.listEntries(tc.params)

			require.Equal(s.T(), f.idsOf(tc.want...), entryIDs(result.Entries))
			require.Nil(s.T(), result.Next)
		})
	}
}

// Test_ListJournalEntries_Postings checks that each entry comes back with all
// its postings, in line order, and with the ledger account of each.
func (s *UnitTestSuite) Test_ListJournalEntries_Postings() {
	f := s.seedJournal()

	result := s.listEntries(lt.ListJournalEntriesParams{From: listAt(1), To: listAt(3)})

	require.Len(s.T(), result.Entries, 2)
	payroll, groceries := result.Entries[0], result.Entries[1]
	require.Equal(s.T(), f.entries[0].ID, payroll.ID)
	require.Len(s.T(), payroll.Postings, 3)
	for n, want := range []struct {
		account *model.LedgerAccount
		amount  int64
		side    model.PostingSide
	}{
		{f.cash, 80, model.PostingDebit},
		{f.tax, 20, model.PostingDebit},
		{f.salary, 100, model.PostingCredit},
	} {
		posting := payroll.Postings[n]
		require.Equal(s.T(), n+1, posting.LineNumber)
		require.Equal(s.T(), payroll.ID, posting.JournalEntryID)
		require.Equal(s.T(), want.account.ID, posting.LedgerAccountID)
		require.True(s.T(), decimal.NewFromInt(want.amount).Equal(posting.Amount))
		require.Equal(s.T(), want.side, posting.Side)
		require.NotNil(s.T(), posting.LedgerAccount)
		require.Equal(s.T(), want.account.Name, posting.LedgerAccount.Name)
	}
	require.Equal(s.T(), f.employer.ID, *payroll.CounterpartyID)
	require.Equal(s.T(), "payroll-1", *payroll.ExternalID)
	require.Equal(s.T(), model.JournalEntryPosted, payroll.Status)

	require.Equal(s.T(), f.entries[1].ID, groceries.ID)
	require.Len(s.T(), groceries.Postings, 2, "the postings of one entry are not mixed with the next one's")
}

// Test_ListJournalEntries_Pages checks that following the cursor lists every
// entry once and in order, in either direction and whatever the page size,
// including when a page ends between two entries of the same instant.
func (s *UnitTestSuite) Test_ListJournalEntries_Pages() {
	f := s.seedJournal()

	tests := map[string]struct {
		params lt.ListJournalEntriesParams
		// pages are the positions in the fixture of the entries of each page.
		pages [][]int
	}{
		"an entry a page": {
			params: lt.ListJournalEntriesParams{Limit: 1},
			pages:  [][]int{{0}, {1}, {2}, {3}, {4}, {5}},
		},
		"a page ending inside an instant": {
			params: lt.ListJournalEntriesParams{Limit: 3},
			pages:  [][]int{{0, 1, 2}, {3, 4, 5}},
		},
		"a page that is exactly the rest": {
			params: lt.ListJournalEntriesParams{Limit: 4},
			pages:  [][]int{{0, 1, 2, 3}, {4, 5}},
		},
		"a page holding everything": {
			params: lt.ListJournalEntriesParams{Limit: 6},
			pages:  [][]int{{0, 1, 2, 3, 4, 5}},
		},
		"an entry a page, newest first": {
			params: lt.ListJournalEntriesParams{Limit: 1, Descending: true},
			pages:  [][]int{{5}, {4}, {3}, {2}, {1}, {0}},
		},
		"a page ending inside an instant, newest first": {
			params: lt.ListJournalEntriesParams{Limit: 3, Descending: true},
			pages:  [][]int{{5, 4, 3}, {2, 1, 0}},
		},
		"a filtered list": {
			params: lt.ListJournalEntriesParams{Limit: 2, LedgerAccountID: f.cash.ID, Statuses: []model.JournalEntryStatus{model.JournalEntryPosted}},
			pages:  [][]int{{0, 1}, {2, 3}},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			params := tc.params

			for n, want := range tc.pages {
				result := s.listEntries(params)

				require.Equal(s.T(), f.idsOf(want...), entryIDs(result.Entries), "page %d", n)
				if n == len(tc.pages)-1 {
					require.Nil(s.T(), result.Next, "the last page has no next")
					continue
				}
				require.NotNil(s.T(), result.Next, "page %d", n)
				last := result.Entries[len(result.Entries)-1]
				require.Equal(s.T(), last.ID, result.Next.ID)
				require.True(s.T(), last.OccurredAt.Equal(result.Next.OccurredAt))
				params.After = result.Next
			}
		})
	}
}

// Test_ListJournalEntries_Validation checks that parameters that cannot select a
// page are rejected.
func (s *UnitTestSuite) Test_ListJournalEntries_Validation() {
	_, err := s.Env.ExecuteActivity(s.ja.ListJournalEntries, lt.ListJournalEntriesParams{Limit: lt.MaxJournalEntriesLimit + 1})

	require.ErrorContains(s.T(), err, "limit")
}

// Test_ListJournalEntries_TiesAreOrderedById checks that entries of the same
// instant are listed by id, in either direction, and that pages can end between
// them. Two instants have two entries each, one stored in the order of its ids
// and the other against it, so that each direction meets a pair stored the wrong
// way round. The date index, which would return them by id anyway, is dropped so
// that only the query can order them.
func (s *UnitTestSuite) Test_ListJournalEntries_TiesAreOrderedById() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	cash := testLedgerAccount(s.T(), s.DB, eur.ID, "cash", model.LedgerAccountAsset)
	salary := testLedgerAccount(s.T(), s.DB, eur.ID, "salary", model.LedgerAccountIncome)
	_, err := s.DB.ExecContext(s.T().Context(), "DROP INDEX ledger.journal_entry_occurred_at_idx")
	require.NoError(s.T(), err)

	// The entries are inserted in this order, with the ids they are listed by.
	stored := []struct {
		id  int64
		day int
	}{{5001, 1}, {5002, 1}, {6002, 2}, {6001, 2}}
	for _, e := range stored {
		temporaltesting.InsertPostedJournalEntry(s.T(), s.DB, &model.JournalEntry{ID: e.id, OccurredAt: listAt(e.day)},
			model.Posting{LedgerAccountID: cash.ID, Amount: decimal.NewFromInt(1), Side: model.PostingDebit},
			model.Posting{LedgerAccountID: salary.ID, Amount: decimal.NewFromInt(1), Side: model.PostingCredit},
		)
	}
	oldestFirst := []int64{5001, 5002, 6001, 6002}
	newestFirst := []int64{6002, 6001, 5002, 5001}

	require.Equal(s.T(), oldestFirst, entryIDs(s.listEntries(lt.ListJournalEntriesParams{}).Entries))
	require.Equal(s.T(), newestFirst, entryIDs(s.listEntries(lt.ListJournalEntriesParams{Descending: true}).Entries))

	for _, descending := range []bool{false, true} {
		params := lt.ListJournalEntriesParams{Limit: 1, Descending: descending}
		var walked []int64
		for {
			page := s.listEntries(params)
			walked = append(walked, entryIDs(page.Entries)...)
			if page.Next == nil {
				break
			}
			params.After = page.Next
		}

		want := oldestFirst
		if descending {
			want = newestFirst
		}
		require.Equal(s.T(), want, walked, "walking the pages, newest first is %t", descending)
	}
}
