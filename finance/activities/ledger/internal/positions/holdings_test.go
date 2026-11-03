package positions

import (
	"context"
	"testing"
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"github.com/uptrace/bun"
)

var negativeBaseDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

type NegativeHoldingsSuite struct {
	temporaltesting.TemporalDBUnitTestSuite

	first, second *model.FinancialAccount
}

func (s *NegativeHoldingsSuite) BeforeTest(_, _ string) {
	sqllib.InitDatabaseT(s.T(), s.DB)

	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	s.first = &model.FinancialAccount{InstitutionID: institution.ID, Name: "First account"}
	temporaltesting.Insert(s.T(), s.DB, s.first)
	s.second = &model.FinancialAccount{InstitutionID: institution.ID, Name: "Second account"}
	temporaltesting.Insert(s.T(), s.DB, s.second)
}

func (s *NegativeHoldingsSuite) AfterTest(_, _ string) { s.DropAllTables() }

func Test_NegativeHoldingsSuite(t *testing.T) {
	suite.Run(t, new(NegativeHoldingsSuite))
}

// move is a movement of units, in the account of the case, on a day. The
// movements of a case are inserted in the order given, which is the order of
// their ids.
type move struct {
	account int
	asset   int
	units   int64
	day     int
	// draft puts the movement in a journal entry that was never posted, so that
	// it does not count.
	draft bool
}

type negative struct {
	account int
	asset   int
	units   int64
	day     int
}

func (s *NegativeHoldingsSuite) Test_NegativeHoldingsTx() {
	tests := map[string]struct {
		moves []move
		// pairs are the positions asked about, as an account and an asset index.
		pairs [][2]int
		want  []negative
	}{
		"never negative": {
			moves: []move{{0, 0, 10, 1, false}, {0, 0, -4, 2, false}, {0, 0, -6, 3, false}},
			pairs: [][2]int{{0, 0}},
		},
		"no movements": {
			pairs: [][2]int{{0, 0}},
		},
		"negative from one point onwards": {
			moves: []move{{0, 0, 10, 1, false}, {0, 0, -15, 2, false}, {0, 0, -5, 3, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -5, 2}},
		},
		"negative and then positive again reports the first point": {
			moves: []move{{0, 0, -3, 1, false}, {0, 0, 10, 2, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -3, 1}},
		},
		"negative twice reports the first point": {
			moves: []move{{0, 0, 2, 1, false}, {0, 0, -5, 2, false}, {0, 0, 10, 3, false}, {0, 0, -9, 4, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -3, 2}},
		},
		"a sale entered before the purchase is judged by date": {
			moves: []move{{0, 0, -10, 2, false}, {0, 0, 10, 1, false}},
			pairs: [][2]int{{0, 0}},
		},
		"several pairs of which only some are negative": {
			moves: []move{
				{0, 0, 5, 1, false}, {0, 0, -6, 2, false},
				{0, 1, 5, 1, false},
				{1, 0, -1, 3, false},
				{1, 1, 2, 1, false}, {1, 1, -2, 2, false},
			},
			pairs: [][2]int{{0, 0}, {0, 1}, {1, 0}, {1, 1}},
			want:  []negative{{0, 0, -1, 2}, {1, 0, -1, 3}},
		},
		"the result is ordered by account and asset whatever the order asked": {
			moves: []move{{1, 1, -1, 1, false}, {1, 0, -2, 1, false}, {0, 1, -3, 1, false}, {0, 0, -4, 1, false}},
			pairs: [][2]int{{1, 1}, {1, 0}, {0, 1}, {0, 0}},
			want:  []negative{{0, 0, -4, 1}, {0, 1, -3, 1}, {1, 0, -2, 1}, {1, 1, -1, 1}},
		},
		"a pair asked twice is reported once": {
			moves: []move{{0, 0, -1, 1, false}},
			pairs: [][2]int{{0, 0}, {0, 0}},
			want:  []negative{{0, 0, -1, 1}},
		},
		"only the pairs asked about are looked at": {
			moves: []move{{0, 0, -1, 1, false}, {0, 1, -1, 1, false}},
			pairs: [][2]int{{0, 1}},
			want:  []negative{{0, 1, -1, 1}},
		},
		"movements of an entry that is not posted do not count": {
			moves: []move{{0, 0, 5, 1, false}, {0, 0, -9, 2, true}},
			pairs: [][2]int{{0, 0}},
		},
		"movements of an entry that is not posted do not hide a negative holding": {
			moves: []move{{0, 0, 5, 1, true}, {0, 0, -1, 2, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -1, 2}},
		},
		"a tie on the date is broken by insertion order, purchase first": {
			moves: []move{{0, 0, 5, 1, false}, {0, 0, -5, 1, false}, {0, 0, -1, 1, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -1, 1}},
		},
		"a tie on the date is broken by insertion order, sale first": {
			moves: []move{{0, 0, -5, 1, false}, {0, 0, 5, 1, false}},
			pairs: [][2]int{{0, 0}},
			want:  []negative{{0, 0, -5, 1}},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			accounts := []*model.FinancialAccount{s.first, s.second}
			assets := make([]*model.Asset, 2)
			for n := range assets {
				assets[n] = &model.Asset{Name: name + string(rune('a'+n)), Kind: model.AssetKindStock}
				temporaltesting.Insert(s.T(), s.DB, assets[n])
			}
			for _, m := range tc.moves {
				movement := &model.PositionMovement{
					FinancialAccountID: accounts[m.account].ID,
					AssetID:            assets[m.asset].ID,
					UnitsDelta:         decimal.NewFromInt(m.units),
					OccurredAt:         negativeBaseDay.AddDate(0, 0, m.day),
				}
				if m.draft {
					entry := &model.JournalEntry{OccurredAt: movement.OccurredAt, Status: model.JournalEntryDraft}
					temporaltesting.Insert(s.T(), s.DB, entry)
					movement.JournalEntryID = &entry.ID
					movement.Kind = model.PositionMovementTrade
				}
				temporaltesting.InsertPositionMovement(s.T(), s.DB, movement)
			}
			pairs := make([]Pair, len(tc.pairs))
			for n, p := range tc.pairs {
				pairs[n] = Pair{FinancialAccountID: accounts[p[0]].ID, AssetID: assets[p[1]].ID}
			}
			var got []lt.NegativeHolding
			err := s.DB.RunInTx(s.T().Context(), nil, func(ctx context.Context, tx bun.Tx) error {
				var err error
				got, err = NegativeHoldingsTx(ctx, tx, pairs)
				return err
			})
			require.NoError(s.T(), err)

			require.Len(s.T(), got, len(tc.want))
			for n, w := range tc.want {
				require.Equal(s.T(), accounts[w.account].ID, got[n].FinancialAccountID)
				require.Equal(s.T(), accounts[w.account].Name, got[n].FinancialAccountName)
				require.Equal(s.T(), assets[w.asset].ID, got[n].AssetID)
				require.Equal(s.T(), assets[w.asset].Name, got[n].AssetName)
				require.True(s.T(), got[n].Units.Equal(decimal.NewFromInt(w.units)), "holds %s units", got[n].Units)
				require.True(s.T(), got[n].At.Equal(negativeBaseDay.AddDate(0, 0, w.day)), "negative at %s", got[n].At)
			}
		})
	}
}
