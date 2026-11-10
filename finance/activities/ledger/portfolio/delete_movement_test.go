package portfolio

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

// seedAdjustment records an adjustment of units under its own key, and returns
// its movement. A non-nil reverses makes it the reversal of that movement.
func (s *UnitTestSuite) seedAdjustment(key string, asset *model.Asset, units int64, day int, reverses *model.PositionMovement) *model.PositionMovement {
	s.T().Helper()

	leg := &model.PositionMovement{FinancialAccountID: s.financialAccount.ID, UnitsDelta: decimal.NewFromInt(units)}
	if reverses != nil {
		leg.ReversalOfMovementID = &reverses.ID
	}
	return s.seedGroup(key, model.PositionMovementAdjustment, asset, day, leg)[0]
}

func (s *UnitTestSuite) movementsExist(movements []*model.PositionMovement) int64 {
	s.T().Helper()

	ids := make([]int64, len(movements))
	for n, movement := range movements {
		ids[n] = movement.ID
	}
	if len(ids) == 0 {
		return 0
	}

	return int64(s.CountWhere[model.PositionMovement]("id IN (?)", bun.List(ids)))
}

// Test_DeletePositionMovement checks that any movement of an entry-less command
// deletes the whole command and the groups that reverse it, and nothing else.
// The cases share the database, so each uses an asset and keys of its own.
func (s *UnitTestSuite) Test_DeletePositionMovement() {
	tests := map[string]struct {
		// seed inserts the movements of the case, and returns the one to delete
		// by, the movements that go, and the ones that stay.
		seed func(asset *model.Asset) (target *model.PositionMovement, gone, kept []*model.PositionMovement)
		// wantUnits is the units the only warning reports, and none means no
		// warning.
		wantUnits *int64
		closed    bool
	}{
		"an adjustment": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			movement := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
			return movement, []*model.PositionMovement{movement}, nil
		}},
		"a transfer, by its first leg": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			legs := s.seedTransfer(asset.Name, asset)
			return legs[0], legs, nil
		}},
		"a transfer, by its second leg": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			legs := s.seedTransfer(asset.Name, asset)
			return legs[1], legs, nil
		}},
		"another command stays": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			target := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
			other := s.seedAdjustment(asset.Name+" other", asset, 4, 2, nil)
			return target, []*model.PositionMovement{target}, []*model.PositionMovement{other}
		}},
		"a command with its reversal": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			original := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
			reversal := s.seedAdjustment(asset.Name+" reversal", asset, -3, 2, original)
			return original, []*model.PositionMovement{original, reversal}, nil
		}},
		"a command with a chain of reversals": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			original := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
			reversal := s.seedAdjustment(asset.Name+" reversal", asset, -3, 2, original)
			again := s.seedAdjustment(asset.Name+" again", asset, 3, 3, reversal)
			return original, []*model.PositionMovement{original, reversal, again}, nil
		}},
		"a reversal alone": {seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			original := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
			reversal := s.seedAdjustment(asset.Name+" reversal", asset, -3, 2, original)
			return reversal, []*model.PositionMovement{reversal}, []*model.PositionMovement{original}
		}},
		"units used since": {
			seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
				purchase := s.seedAdjustment(asset.Name, asset, 10, 1, nil)
				sale := s.seedAdjustment(asset.Name+" sale", asset, -10, 5, nil)
				return purchase, []*model.PositionMovement{purchase}, []*model.PositionMovement{sale}
			},
			wantUnits: ptr.Int64(-10),
		},
		"a closed financial account": {
			closed: true,
			seed: func(asset *model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
				movement := s.seedAdjustment(asset.Name, asset, 3, 1, nil)
				return movement, []*model.PositionMovement{movement}, nil
			},
		},
		"a movement that does not exist": {seed: func(*model.Asset) (*model.PositionMovement, []*model.PositionMovement, []*model.PositionMovement) {
			return &model.PositionMovement{ID: 987654}, nil, nil
		}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			target, gone, kept := tc.seed(asset)
			if tc.closed {
				closedAt := time.Now().UTC()
				s.financialAccount.ClosedAt = &closedAt
				_, err := s.DB.NewUpdate().Model(s.financialAccount).Column("closed_at").WherePK().Exec(s.T().Context())
				require.NoError(s.T(), err)
				s.T().Cleanup(func() {
					s.financialAccount.ClosedAt = nil
					_, err := s.DB.NewUpdate().Model(s.financialAccount).Set("closed_at = NULL").WherePK().Exec(s.T().Context())
					require.NoError(s.T(), err)
				})
			}

			activityResult, err := s.Env.ExecuteActivity(s.pa.DeletePositionMovement, lt.DeletePositionMovementInput{MovementID: target.ID})

			require.NoError(s.T(), err)
			var result lt.DeletePositionMovementResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Zero(s.T(), s.movementsExist(gone))
			require.EqualValues(s.T(), len(kept), s.movementsExist(kept))
			if tc.wantUnits == nil {
				require.Empty(s.T(), result.Warnings)
				return
			}
			require.Len(s.T(), result.Warnings, 1)
			warning := result.Warnings[0]
			require.Equal(s.T(), s.financialAccount.ID, warning.FinancialAccountID)
			require.Equal(s.T(), asset.ID, warning.AssetID)
			require.True(s.T(), warning.Units.Equal(decimal.NewFromInt(*tc.wantUnits)), "holds %s units", warning.Units)
		})
	}
}

// Test_DeletePositionMovement_Rejected checks that a movement that cannot be
// deleted leaves no row behind.
func (s *UnitTestSuite) Test_DeletePositionMovement_Rejected() {
	tests := map[string]struct {
		input   func(asset *model.Asset) lt.DeletePositionMovementInput
		failure string
	}{
		"no movement": {
			failure: "validation error",
			input:   func(*model.Asset) lt.DeletePositionMovementInput { return lt.DeletePositionMovementInput{} },
		},
		"a movement of a journal entry": {
			failure: "delete the journal entry",
			input: func(asset *model.Asset) lt.DeletePositionMovementInput {
				movement := &model.PositionMovement{FinancialAccountID: s.financialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), Kind: model.PositionMovementTrade}
				temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB,
					&model.JournalEntry{OccurredAt: movementDay},
					[]model.Posting{
						{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
						{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
					},
					[]*model.PositionMovement{movement})
				return lt.DeletePositionMovementInput{MovementID: movement.ID}
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			input := tc.input(asset)
			rows := s.Count[model.PositionMovement]()

			_, err := s.Env.ExecuteActivity(s.pa.DeletePositionMovement, input)

			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), rows, s.Count[model.PositionMovement](), "a rejected delete must leave nothing behind")
		})
	}
}

// Test_DeletePositionMovement_Idempotent checks that deleting again succeeds and
// changes nothing, and that the key of the command is free afterwards.
func (s *UnitTestSuite) Test_DeletePositionMovement_Idempotent() {
	purchase := s.seedAdjustment("purchase", s.asset, 10, 1, nil)
	sale := s.seedAdjustment("sale", s.asset, -10, 5, nil)
	input := lt.DeletePositionMovementInput{MovementID: purchase.ID}

	first, err := s.Env.ExecuteActivity(s.pa.DeletePositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.DeletePositionMovementResult
	require.NoError(s.T(), first.Get(&created))
	require.Len(s.T(), created.Warnings, 1)
	require.Equal(s.T(), 1, s.Count[model.PositionMovement]())

	second, err := s.Env.ExecuteActivity(s.pa.DeletePositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.DeletePositionMovementResult
	require.NoError(s.T(), second.Get(&retried))
	require.Empty(s.T(), retried.Warnings, "a delete that finds nothing touches no position")
	require.Equal(s.T(), 1, s.Count[model.PositionMovement]())
	require.EqualValues(s.T(), 1, s.movementsExist([]*model.PositionMovement{sale}))

	// The insert fails on the unique key unless the delete freed it.
	s.seedAdjustment("purchase", s.asset, 7, 1, nil)
}
