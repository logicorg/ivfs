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

// seedGroup inserts the legs of a command without a journal entry, as movements
// that were already there, under one key and numbered in leg order.
func (s *UnitTestSuite) seedGroup(key string, kind model.PositionMovementKind, asset *model.Asset, day int, legs ...*model.PositionMovement) []*model.PositionMovement {
	s.T().Helper()

	for n, leg := range legs {
		leg.Kind, leg.AssetID = kind, asset.ID
		leg.OccurredAt = movementDay.AddDate(0, 0, day)
		leg.IdempotencyKey, leg.LineNumber = ptr.NewPtr(key), n+1
		leg.ExternalID = ptr.NewPtr("broker-ref")
		temporaltesting.Insert(s.T(), s.DB, leg)
	}
	return legs
}

// seedTransfer moves 4 of the 10 units the first account holds to the second,
// with its cost basis, and returns the two legs.
func (s *UnitTestSuite) seedTransfer(key string, asset *model.Asset) []*model.PositionMovement {
	s.T().Helper()

	s.seedUnits(s.financialAccount, asset, 10, 1)
	return s.seedGroup(key, model.PositionMovementTransfer, asset, 5,
		&model.PositionMovement{
			FinancialAccountID: s.financialAccount.ID, UnitsDelta: decimal.NewFromInt(-4),
			CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(-40)), CostBasisCurrencyID: ptr.Int64(s.eur.ID),
		},
		&model.PositionMovement{
			FinancialAccountID: s.otherAccount.ID, UnitsDelta: decimal.NewFromInt(4),
			CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(40)), CostBasisCurrencyID: ptr.Int64(s.eur.ID),
		},
	)
}

// Test_ReversePositionMovement checks that reversing any movement of a command
// reverses the whole command with the opposite movements, linked to the ones
// they cancel, and puts the units held back where they were.
func (s *UnitTestSuite) Test_ReversePositionMovement() {
	reversedAt := movementDay.AddDate(0, 0, 10)

	tests := map[string]struct {
		// seed inserts the command under key and returns its legs.
		seed func(asset *model.Asset, key string) []*model.PositionMovement
		// reverse is the position in the group of the movement that is reversed.
		reverse int
		// wantFirst and wantSecond are what the first and the second account hold
		// after the reversal.
		wantFirst, wantSecond int64
	}{
		"a transfer, by its first leg": {
			seed:    func(asset *model.Asset, key string) []*model.PositionMovement { return s.seedTransfer(key, asset) },
			reverse: 0, wantFirst: 10, wantSecond: 0,
		},
		"a transfer, by its second leg": {
			seed:    func(asset *model.Asset, key string) []*model.PositionMovement { return s.seedTransfer(key, asset) },
			reverse: 1, wantFirst: 10, wantSecond: 0,
		},
		"an adjustment": {
			seed: func(asset *model.Asset, key string) []*model.PositionMovement {
				s.seedUnits(s.financialAccount, asset, 10, 1)
				return s.seedGroup(key, model.PositionMovementAdjustment, asset, 5,
					&model.PositionMovement{FinancialAccountID: s.financialAccount.ID, UnitsDelta: decimal.NewFromInt(-3), Description: ptr.NewPtr("correction")})
			},
			wantFirst: 10,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			originals := tc.seed(asset, name)
			reversalKey := name + " reversal"

			activityResult, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, lt.ReversePositionMovementInput{
				MovementID: originals[tc.reverse].ID, IdempotencyKey: reversalKey, OccurredAt: reversedAt,
			})

			require.NoError(s.T(), err)
			var result lt.ReversePositionMovementResult
			require.NoError(s.T(), activityResult.Get(&result))
			reversals := s.movementsWithKey(reversalKey)
			require.Len(s.T(), reversals, len(originals))
			require.Len(s.T(), result.PositionMovementIDs, len(originals))

			for n, reversal := range reversals {
				original := originals[n]
				require.Equal(s.T(), result.PositionMovementIDs[n], reversal.ID, "ids follow the movements they reverse")
				require.Equal(s.T(), original.ID, *reversal.ReversalOfMovementID)
				require.Equal(s.T(), original.Kind, reversal.Kind)
				require.Equal(s.T(), original.FinancialAccountID, reversal.FinancialAccountID)
				require.Equal(s.T(), original.AssetID, reversal.AssetID)
				require.Equal(s.T(), n+1, reversal.LineNumber)
				require.Nil(s.T(), reversal.JournalEntryID)
				require.Nil(s.T(), reversal.ExternalID, "the external id of the original is not copied")
				require.True(s.T(), original.UnitsDelta.Neg().Equal(reversal.UnitsDelta))
				require.True(s.T(), reversedAt.Equal(reversal.OccurredAt), "a reversal is dated when it is made, not as the original")
				require.Equal(s.T(), original.CostBasisCurrencyID, reversal.CostBasisCurrencyID)
				if original.CostBasisAmount == nil {
					require.Nil(s.T(), reversal.CostBasisAmount)
					continue
				}
				require.True(s.T(), original.CostBasisAmount.Neg().Equal(*reversal.CostBasisAmount))
			}

			require.True(s.T(), s.unitsHeld(s.financialAccount, asset).Equal(decimal.NewFromInt(tc.wantFirst)))
			require.True(s.T(), s.unitsHeld(s.otherAccount, asset).Equal(decimal.NewFromInt(tc.wantSecond)))
		})
	}
}

// Test_ReversePositionMovement_DefaultDate checks that a reversal without a date
// is dated when it is made.
func (s *UnitTestSuite) Test_ReversePositionMovement_DefaultDate() {
	originals := s.seedTransfer("transfer-1", s.asset)
	before := time.Now().UTC()

	_, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: "reversal-1"})

	require.NoError(s.T(), err)
	for _, reversal := range s.movementsWithKey("reversal-1") {
		require.WithinDuration(s.T(), before, reversal.OccurredAt, time.Minute)
	}
}

// Test_ReversePositionMovement_Idempotent checks that a retry returns the
// reversing movements of the first call and changes nothing.
func (s *UnitTestSuite) Test_ReversePositionMovement_Idempotent() {
	originals := s.seedTransfer("transfer-1", s.asset)
	input := lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: "reversal-1", OccurredAt: movementDay.AddDate(0, 0, 10)}

	first, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.ReversePositionMovementResult
	require.NoError(s.T(), first.Get(&created))
	rows := s.countMovements()

	// The retry comes from the other leg and a later date, which is not compared.
	input.MovementID, input.OccurredAt = originals[1].ID, movementDay.AddDate(0, 0, 20)
	second, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.ReversePositionMovementResult
	require.NoError(s.T(), second.Get(&retried))

	require.Equal(s.T(), created.PositionMovementIDs, retried.PositionMovementIDs)
	require.Empty(s.T(), created.Warnings)
	require.Empty(s.T(), retried.Warnings)
	require.Equal(s.T(), rows, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.financialAccount, s.asset).Equal(decimal.NewFromInt(10)))
	require.True(s.T(), s.unitsHeld(s.otherAccount, s.asset).Equal(decimal.Zero))
}

// seedUsedAdjustment gives the first account 10 units by an adjustment, and
// sells 8 of them since, so that reversing the adjustment leaves -8.
func (s *UnitTestSuite) seedUsedAdjustment(asset *model.Asset) []*model.PositionMovement {
	s.T().Helper()

	adjustment := s.seedGroup(asset.Name+" adjustment", model.PositionMovementAdjustment, asset, 1,
		&model.PositionMovement{FinancialAccountID: s.financialAccount.ID, UnitsDelta: decimal.NewFromInt(10), Description: ptr.NewPtr("correction")})
	s.seedUnits(s.financialAccount, asset, -8, 2)
	return adjustment
}

// Test_ReversePositionMovement_UnitsUsedSince checks that a movement whose units
// were used since is reversed anyway, and that the reversal warns about the
// position it leaves negative, at the date of the reversal.
func (s *UnitTestSuite) Test_ReversePositionMovement_UnitsUsedSince() {
	asset := &model.Asset{Name: "used", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	adjustment := s.seedUsedAdjustment(asset)
	rows := s.countMovements()

	activityResult, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, lt.ReversePositionMovementInput{
		MovementID: adjustment[0].ID, IdempotencyKey: "used-reversal", OccurredAt: movementDay.AddDate(0, 0, 10),
	})

	require.NoError(s.T(), err)
	var result lt.ReversePositionMovementResult
	require.NoError(s.T(), activityResult.Get(&result))
	s.requireWarnings(result.Warnings, asset, expectedWarning{s.financialAccount, negativeAt{-8, 10}})
	require.Len(s.T(), result.PositionMovementIDs, 1)
	require.Equal(s.T(), rows+1, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.financialAccount, asset).Equal(decimal.NewFromInt(-8)))
}

// Test_ReversePositionMovement_IdempotentWarnings checks that a retry returns
// the warnings as they are now, so none once the units have been backfilled.
func (s *UnitTestSuite) Test_ReversePositionMovement_IdempotentWarnings() {
	adjustment := s.seedUsedAdjustment(s.asset)
	input := lt.ReversePositionMovementInput{MovementID: adjustment[0].ID, IdempotencyKey: "warning-reversal", OccurredAt: movementDay.AddDate(0, 0, 10)}

	first, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.ReversePositionMovementResult
	require.NoError(s.T(), first.Get(&created))
	s.requireWarnings(created.Warnings, s.asset, expectedWarning{s.financialAccount, negativeAt{-8, 10}})
	rows := s.countMovements()

	second, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.ReversePositionMovementResult
	require.NoError(s.T(), second.Get(&retried))
	require.Equal(s.T(), created, retried)

	s.seedUnits(s.financialAccount, s.asset, 8, 0)

	third, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)
	require.NoError(s.T(), err)
	var afterBackfill lt.ReversePositionMovementResult
	require.NoError(s.T(), third.Get(&afterBackfill))
	require.Equal(s.T(), created.PositionMovementIDs, afterBackfill.PositionMovementIDs)
	require.Empty(s.T(), afterBackfill.Warnings)
	require.Equal(s.T(), rows+1, s.countMovements())
}

// Test_ReversePositionMovement_Rejected checks that a movement that cannot be
// reversed leaves no row behind.
func (s *UnitTestSuite) Test_ReversePositionMovement_Rejected() {
	tests := map[string]struct {
		// input seeds what the case needs and returns the reversal to attempt. The
		// keys it uses start with the name of the asset, which is the name of the
		// case, since the cases share the database.
		input func(asset *model.Asset) lt.ReversePositionMovementInput
		// failure is what the error has to contain.
		failure string
	}{
		"a movement that does not exist": {
			failure: "PositionMovement",
			input: func(*model.Asset) lt.ReversePositionMovementInput {
				return lt.ReversePositionMovementInput{MovementID: 999999, IdempotencyKey: "missing reversal"}
			},
		},
		"a movement of a journal entry": {
			failure: "reverse the journal entry",
			input: func(asset *model.Asset) lt.ReversePositionMovementInput {
				movement := &model.PositionMovement{FinancialAccountID: s.financialAccount.ID, AssetID: asset.ID, UnitsDelta: decimal.NewFromInt(5), Kind: model.PositionMovementTrade}
				temporaltesting.InsertPostedJournalEntryWithMovements(s.T(), s.DB,
					&model.JournalEntry{OccurredAt: movementDay},
					[]model.Posting{
						{LedgerAccountID: s.offset.ID, Amount: decimal.NewFromInt(50), Side: model.PostingDebit},
						{LedgerAccountID: s.cash.ID, Amount: decimal.NewFromInt(50), Side: model.PostingCredit},
					},
					[]*model.PositionMovement{movement})
				return lt.ReversePositionMovementInput{MovementID: movement.ID, IdempotencyKey: asset.Name + " reversal"}
			},
		},
		"a command that was already reversed": {
			failure: "already been reversed",
			input: func(asset *model.Asset) lt.ReversePositionMovementInput {
				originals := s.seedTransfer(asset.Name+" transfer", asset)
				for n, original := range originals {
					temporaltesting.Insert(s.T(), s.DB, &model.PositionMovement{
						Kind: original.Kind, FinancialAccountID: original.FinancialAccountID, AssetID: asset.ID,
						UnitsDelta: original.UnitsDelta.Neg(), OccurredAt: movementDay.AddDate(0, 0, 6),
						IdempotencyKey: ptr.NewPtr(asset.Name + " first reversal"), LineNumber: n + 1, ReversalOfMovementID: &original.ID,
					})
				}
				return lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: asset.Name + " second reversal"}
			},
		},
		"a key that belongs to another command": {
			failure: "not a reversal",
			input: func(asset *model.Asset) lt.ReversePositionMovementInput {
				originals := s.seedTransfer(asset.Name+" transfer", asset)
				other := s.seedGroup(asset.Name+" other command", model.PositionMovementAdjustment, asset, 6,
					&model.PositionMovement{FinancialAccountID: s.financialAccount.ID, UnitsDelta: decimal.NewFromInt(1), Description: ptr.NewPtr("correction")})
				return lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: *other[0].IdempotencyKey}
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			input := tc.input(asset)
			rows := s.countMovements()

			_, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, input)

			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), rows, s.countMovements(), "a rejected reversal must leave nothing behind")
		})
	}
}

// Test_ReversePositionMovement_ClosedAccount checks that a movement can still be
// reversed once its financial account is closed, since a reversal corrects
// history rather than adding to it.
func (s *UnitTestSuite) Test_ReversePositionMovement_ClosedAccount() {
	originals := s.seedTransfer("transfer-1", s.asset)
	s.closeAccount(s.otherAccount)

	_, err := s.Env.ExecuteActivity(s.pa.ReversePositionMovement, lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: "reversal-1"})

	require.NoError(s.T(), err)
	require.True(s.T(), s.unitsHeld(s.otherAccount, s.asset).Equal(decimal.Zero))
}

// Test_ReversePositionMovement_WaitsForConcurrentWriter checks that a reversal
// waits for a transaction that has locked one of its accounts, and then sees
// what that transaction wrote. A reversal of the same command is held open
// here, so without the lock the second one would not see it, and would fail on
// the unique constraint with an error that is retried instead of a conflict.
func (s *UnitTestSuite) Test_ReversePositionMovement_WaitsForConcurrentWriter() {
	originals := s.seedTransfer("transfer-1", s.asset)
	commit := s.holdAccount(s.financialAccount, &model.PositionMovement{
		Kind: model.PositionMovementTransfer, FinancialAccountID: s.financialAccount.ID, AssetID: s.asset.ID,
		UnitsDelta: decimal.NewFromInt(4), OccurredAt: movementDay.AddDate(0, 0, 6),
		IdempotencyKey: ptr.NewPtr("racing-reversal"), LineNumber: 1, ReversalOfMovementID: &originals[0].ID,
	})

	done := make(chan error, 1)
	go func() {
		_, err := s.pa.ReversePositionMovement(s.T().Context(), lt.ReversePositionMovementInput{MovementID: originals[0].ID, IdempotencyKey: "reversal-2"})
		done <- err
	}()
	select {
	case <-done:
		s.T().Fatal("the reversal finished without waiting for the transaction holding the account")
	case <-time.After(500 * time.Millisecond):
	}

	commit()
	require.ErrorContains(s.T(), <-done, "already been reversed")
}
