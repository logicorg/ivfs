package portfolio

import (
	"time"

	lt "github.com/logicorg/ivfs/finance/common/types/ledgertypes"
	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_PositionAt checks which movements count towards the units held: one
// without a journal entry and one of a posted entry do, while one of a draft
// never does.
func (s *UnitTestSuite) Test_PositionAt() {
	january := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	february := january.AddDate(0, 1, 0)

	newMovement := func(units int64, occurredAt time.Time) *model.PositionMovement {
		return &model.PositionMovement{
			FinancialAccountID: s.financialAccount.ID,
			AssetID:            s.asset.ID,
			UnitsDelta:         decimal.NewFromInt(units),
			OccurredAt:         occurredAt,
		}
	}

	temporaltesting.InsertPositionMovement(s.T(), s.DB, newMovement(10, january))

	posted := &model.JournalEntry{OccurredAt: february}
	temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB, posted,
		[]model.Posting{
			{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(5), Side: model.PostingDebit},
			{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(5), Side: model.PostingCredit},
		},
		[]*model.PositionMovement{newMovement(5, february)},
	)

	draft := &model.JournalEntry{OccurredAt: february, Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, draft)
	draftMovement := newMovement(100, february)
	draftMovement.JournalEntryID = &draft.ID
	temporaltesting.InsertPositionMovement(s.T(), s.DB, draftMovement)

	tests := map[string]struct {
		at       time.Time
		expected int64
	}{
		"now":                               {expected: 15},
		"before the first movement":         {at: january.Add(-time.Hour), expected: 0},
		"at the first movement is included": {at: january, expected: 10},
		"between the movements":             {at: january.AddDate(0, 0, 15), expected: 10},
		"at the posted entry is included":   {at: february, expected: 15},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			activityResult, err := s.Env.ExecuteActivity(s.pa.PositionAt, lt.PositionAtParams{
				FinancialAccountID: s.financialAccount.ID,
				AssetID:            s.asset.ID,
				At:                 tc.at,
			})
			require.NoError(s.T(), err)

			var position lt.PositionAtResult
			require.NoError(s.T(), activityResult.Get(&position))
			require.True(s.T(), position.Units.Equal(decimal.NewFromInt(tc.expected)), "got %s units", position.Units)
		})
	}
}

func (s *UnitTestSuite) Test_PositionAt_NoMovements() {
	activityResult, err := s.Env.ExecuteActivity(s.pa.PositionAt, lt.PositionAtParams{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            s.asset.ID,
	})
	require.NoError(s.T(), err)

	var position lt.PositionAtResult
	require.NoError(s.T(), activityResult.Get(&position))
	require.True(s.T(), position.Units.IsZero())
	require.Equal(s.T(), s.financialAccount.ID, position.FinancialAccountID)
	require.Equal(s.T(), s.asset.ID, position.AssetID)
}

func (s *UnitTestSuite) Test_PositionAt_InvalidParams() {
	tests := map[string]lt.PositionAtParams{
		"missing financial account": {AssetID: 1},
		"missing asset":             {FinancialAccountID: 1},
	}

	for name, params := range tests {
		s.Run(name, func() {
			_, err := s.Env.ExecuteActivity(s.pa.PositionAt, params)
			require.Error(s.T(), err)
		})
	}
}
