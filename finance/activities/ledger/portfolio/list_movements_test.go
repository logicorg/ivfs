package portfolio

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func historyDay(day int) time.Time { return movementDay.AddDate(0, 0, day) }

// seedDraft inserts a movement of a draft entry, which does not count towards
// holdings.
func (s *UnitTestSuite) seedDraft(asset *model.Asset) {
	s.T().Helper()

	entry := &model.JournalEntry{OccurredAt: historyDay(2), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)
	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{
		JournalEntryID: &entry.ID, FinancialAccountID: s.financialAccount.ID, AssetID: asset.ID,
		UnitsDelta: decimal.NewFromInt(100), OccurredAt: entry.OccurredAt,
	})
}

// seedHistory inserts the history of asset that the tests read, and returns the
// movements that count towards holdings in the order the history lists them:
//
//	0  day 1  first account   adjustment  +10
//	1  day 2  first account   trade       +5   (posted entry)
//	2  day 3  first account   split       +15
//	3  day 3  second account  split       +3   (same instant as 2)
//	4  day 4  first account   transfer    -4
//	5  day 4  second account  transfer    +4
//	6  day 5  first account   transfer    +4   (back from 5)
//	7  day 5  second account  transfer    -4   (back to 4)
//
// It also inserts what the history must leave out: the movement of a draft
// entry and a movement of another asset.
func (s *UnitTestSuite) seedHistory(asset *model.Asset) []*model.PositionMovement {
	s.T().Helper()

	first, second := s.financialAccount, s.otherAccount
	leg := func(account *model.FinancialAccount, units int64) *model.PositionMovement {
		return &model.PositionMovement{FinancialAccountID: account.ID, UnitsDelta: decimal.NewFromInt(units), Description: ptr.NewPtr("history")}
	}

	adjustment := s.seedGroup(asset.Name+" adjustment", model.PositionMovementAdjustment, asset, 1, leg(first, 10))

	trade := &model.PositionMovement{FinancialAccountID: first.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5)}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB,
		&model.JournalEntry{OccurredAt: historyDay(2)},
		[]model.Posting{
			{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
			{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
		},
		[]*model.PositionMovement{trade})

	split := s.seedGroup(asset.Name+" split", model.PositionMovementSplit, asset, 3, leg(first, 15), leg(second, 3))
	transfer := s.seedGroup(asset.Name+" transfer", model.PositionMovementTransfer, asset, 4, leg(first, -4), leg(second, 4))

	back := s.seedGroup(asset.Name+" back", model.PositionMovementTransfer, asset, 5, leg(first, 4), leg(second, -4))

	s.seedDraft(asset)
	other := &model.Asset{Name: asset.Name + " other", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, other)
	s.seedGroup(asset.Name+" other asset", model.PositionMovementAdjustment, other, 1, leg(first, 7))

	return []*model.PositionMovement{adjustment[0], trade, split[0], split[1], transfer[0], transfer[1], back[0], back[1]}
}

func movementIDs(movements []model.PositionMovement) []int64 {
	ids := make([]int64, len(movements))
	for n := range movements {
		ids[n] = movements[n].ID
	}
	return ids
}

// idsOf returns the ids of the movements at the given positions of the history.
func idsOf(history []*model.PositionMovement, positions ...int) []int64 {
	ids := make([]int64, len(positions))
	for n, position := range positions {
		ids[n] = history[position].ID
	}
	return ids
}

func (s *UnitTestSuite) listMovements(params lt.ListPositionMovementsParams) lt.ListPositionMovementsResult {
	s.T().Helper()

	activityResult, err := s.Env.ExecuteActivity(s.pa.ListPositionMovements, params)
	require.NoError(s.T(), err)
	var result lt.ListPositionMovementsResult
	require.NoError(s.T(), activityResult.Get(&result))
	return result
}

// Test_ListPositionMovements checks each filter of the history, and that only
// the movements that count towards holdings are listed, oldest first.
func (s *UnitTestSuite) Test_ListPositionMovements() {
	tests := map[string]struct {
		// filter narrows the history of asset, which every case has to select.
		filter func(params *lt.ListPositionMovementsParams)
		// want are the positions in the history of the movements listed.
		want []int
	}{
		"everything, oldest first": {
			filter: func(*lt.ListPositionMovementsParams) {},
			want:   []int{0, 1, 2, 3, 4, 5, 6, 7},
		},
		"one financial account": {
			filter: func(p *lt.ListPositionMovementsParams) { p.FinancialAccountID = s.financialAccount.ID },
			want:   []int{0, 1, 2, 4, 6},
		},
		"the other financial account": {
			filter: func(p *lt.ListPositionMovementsParams) { p.FinancialAccountID = s.otherAccount.ID },
			want:   []int{3, 5, 7},
		},
		"several kinds": {
			filter: func(p *lt.ListPositionMovementsParams) {
				p.Kinds = []model.PositionMovementKind{model.PositionMovementSplit, model.PositionMovementTransfer}
			},
			want: []int{2, 3, 4, 5, 6, 7},
		},
		"the kind of an entry": {
			filter: func(p *lt.ListPositionMovementsParams) {
				p.Kinds = []model.PositionMovementKind{model.PositionMovementTrade}
			},
			want: []int{1},
		},
		"a range includes its start": {
			filter: func(p *lt.ListPositionMovementsParams) { p.From = historyDay(3) },
			want:   []int{2, 3, 4, 5, 6, 7},
		},
		"a range excludes its end": {
			filter: func(p *lt.ListPositionMovementsParams) { p.To = historyDay(3) },
			want:   []int{0, 1},
		},
		"a range between two dates": {
			filter: func(p *lt.ListPositionMovementsParams) { p.From, p.To = historyDay(2), historyDay(4) },
			want:   []int{1, 2, 3},
		},
		"an account, kinds and a range together": {
			filter: func(p *lt.ListPositionMovementsParams) {
				p.FinancialAccountID = s.financialAccount.ID
				p.Kinds = []model.PositionMovementKind{model.PositionMovementSplit, model.PositionMovementTransfer}
				p.From, p.To = historyDay(3), historyDay(5)
			},
			want: []int{2, 4},
		},
		"a range after the history": {
			filter: func(p *lt.ListPositionMovementsParams) { p.From = historyDay(10) },
			want:   []int{},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			history := s.seedHistory(asset)
			params := lt.ListPositionMovementsParams{AssetID: asset.ID}
			tc.filter(&params)

			result := s.listMovements(params)

			require.Equal(s.T(), idsOf(history, tc.want...), movementIDs(result.Movements))
			require.Nil(s.T(), result.Next)
		})
	}
}

// Test_ListPositionMovements_Pages checks that following the cursor lists every
// movement once and in order, whatever the page size, including when a page ends
// between two movements of the same instant.
func (s *UnitTestSuite) Test_ListPositionMovements_Pages() {
	tests := map[string]struct {
		limit int
		// pages are the positions in the history of the movements of each page.
		pages [][]int
	}{
		"a movement a page":                {limit: 1, pages: [][]int{{0}, {1}, {2}, {3}, {4}, {5}, {6}, {7}}},
		"two movements a page":             {limit: 2, pages: [][]int{{0, 1}, {2, 3}, {4, 5}, {6, 7}}},
		"a page ending inside an instant":  {limit: 3, pages: [][]int{{0, 1, 2}, {3, 4, 5}, {6, 7}}},
		"a page that is exactly the rest":  {limit: 4, pages: [][]int{{0, 1, 2, 3}, {4, 5, 6, 7}}},
		"a page holding the whole history": {limit: 8, pages: [][]int{{0, 1, 2, 3, 4, 5, 6, 7}}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			history := s.seedHistory(asset)
			params := lt.ListPositionMovementsParams{AssetID: asset.ID, Limit: tc.limit}

			for n, want := range tc.pages {
				result := s.listMovements(params)

				require.Equal(s.T(), idsOf(history, want...), movementIDs(result.Movements), "page %d", n)
				if n == len(tc.pages)-1 {
					require.Nil(s.T(), result.Next, "the last page has no next")
					continue
				}
				require.NotNil(s.T(), result.Next, "page %d", n)
				last := result.Movements[len(result.Movements)-1]
				require.Equal(s.T(), last.ID, result.Next.ID)
				require.True(s.T(), last.OccurredAt.Equal(result.Next.OccurredAt))
				require.True(s.T(), last.CreatedAt.Equal(result.Next.CreatedAt))
				params.After = result.Next
			}
		})
	}
}

// Test_ListPositionMovements_RecordingOrder checks that movements of the same
// instant are listed by when they were recorded and not by id, and that the
// cursor follows that order. A movement written again with its entry has a
// later id and keeps when it was recorded (ADR 0017).
func (s *UnitTestSuite) Test_ListPositionMovements_RecordingOrder() {
	asset := &model.Asset{Name: "Recording order", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	recorded := historyDay(30)
	movement := func(units int64, createdAt time.Time) *model.PositionMovement {
		m := &model.PositionMovement{
			FinancialAccountID: s.financialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(units),
			OccurredAt: historyDay(1), CreatedAt: createdAt,
		}
		temporaltesting.InsertPositionMovement(s.T(), s.DB, m)
		return m
	}
	sale := movement(-5, recorded.Add(time.Minute))
	buy := movement(5, recorded)
	later := movement(1, recorded.Add(2*time.Minute))
	want := []int64{buy.ID, sale.ID, later.ID}

	whole := s.listMovements(lt.ListPositionMovementsParams{AssetID: asset.ID})
	require.Equal(s.T(), want, movementIDs(whole.Movements))

	params := lt.ListPositionMovementsParams{AssetID: asset.ID, Limit: 1}
	var paged []int64
	for {
		page := s.listMovements(params)
		paged = append(paged, movementIDs(page.Movements)...)
		if page.Next == nil {
			break
		}
		params.After = page.Next
	}
	require.Equal(s.T(), want, paged)
}

// Test_ListPositionMovements_MatchesPositionAt checks that the sum of the
// history up to a date is the number of units PositionAt reports for it.
func (s *UnitTestSuite) Test_ListPositionMovements_MatchesPositionAt() {
	s.seedHistory(s.asset)

	for day := 0; day <= 6; day++ {
		// An hour after the day, since PositionAt includes what happened at its
		// date while the history excludes it.
		at := historyDay(day).Add(time.Hour)

		for _, account := range []*model.FinancialAccount{s.financialAccount, s.otherAccount} {
			result := s.listMovements(lt.ListPositionMovementsParams{
				FinancialAccountID: account.ID, AssetID: s.asset.ID, To: at, Limit: lt.MaxPositionMovementsLimit,
			})

			sum := decimal.Zero
			for _, movement := range result.Movements {
				sum = sum.Add(movement.UnitsDelta)
			}
			position, err := s.Env.ExecuteActivity(s.pa.PositionAt, lt.PositionAtParams{FinancialAccountID: account.ID, AssetID: s.asset.ID, At: at})
			require.NoError(s.T(), err)
			var held lt.PositionAtResult
			require.NoError(s.T(), position.Get(&held))

			require.True(s.T(), sum.Equal(held.Units), "day %d, account %d: history sums to %s, position is %s", day, account.ID, sum, held.Units)
		}
	}
}

// Test_ListPositionMovements_Validation checks that parameters that cannot select
// a history are rejected.
func (s *UnitTestSuite) Test_ListPositionMovements_Validation() {
	_, err := s.Env.ExecuteActivity(s.pa.ListPositionMovements, lt.ListPositionMovementsParams{Limit: lt.MaxPositionMovementsLimit + 1})

	require.ErrorContains(s.T(), err, "limit")
}
