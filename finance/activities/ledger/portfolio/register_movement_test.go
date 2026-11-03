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

var movementDay = time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

// seedUnits gives account units of asset on the given day, as a movement that
// was already there.
func (s *UnitTestSuite) seedUnits(account *model.FinancialAccount, asset *model.Asset, units int64, day int) {
	s.T().Helper()

	temporaltesting.InsertPositionMovement(s.T(), s.DB, &model.PositionMovement{
		FinancialAccountID: account.ID,
		AssetID:            asset.ID,
		UnitsDelta:         decimal.NewFromInt(units),
		OccurredAt:         movementDay.AddDate(0, 0, day),
	})
}

func (s *UnitTestSuite) unitsHeld(account *model.FinancialAccount, asset *model.Asset) decimal.Decimal {
	s.T().Helper()

	position, err := s.Env.ExecuteActivity(s.pa.PositionAt, lt.PositionAtParams{FinancialAccountID: account.ID, AssetID: asset.ID})
	require.NoError(s.T(), err)
	var result lt.PositionAtResult
	require.NoError(s.T(), position.Get(&result))
	return result.Units
}

func (s *UnitTestSuite) countMovements() int {
	s.T().Helper()

	count, err := s.DB.NewSelect().Model((*model.PositionMovement)(nil)).Count(s.T().Context())
	require.NoError(s.T(), err)
	return count
}

func (s *UnitTestSuite) movementsWithKey(key string) []model.PositionMovement {
	s.T().Helper()

	var movements []model.PositionMovement
	err := s.DB.NewSelect().Model(&movements).Where("pm.idempotency_key = ?", key).OrderExpr("pm.line_number").Scan(s.T().Context())
	require.NoError(s.T(), err)
	return movements
}

func movementLeg(account *model.FinancialAccount, units int64) lt.PositionMovementLeg {
	return lt.PositionMovementLeg{FinancialAccountID: account.ID, UnitsDelta: decimal.NewFromInt(units)}
}

// adjustment is a movement of one leg, which has the fewest rules.
func (s *UnitTestSuite) adjustment(key string, asset *model.Asset, units int64) lt.RegisterPositionMovementInput {
	return lt.RegisterPositionMovementInput{
		IdempotencyKey: key,
		Kind:           model.PositionMovementAdjustment,
		AssetID:        asset.ID,
		OccurredAt:     movementDay.AddDate(0, 0, 5),
		Description:    ptr.NewPtr("broker correction"),
		Legs:           []lt.PositionMovementLeg{movementLeg(s.financialAccount, units)},
	}
}

// transfer moves units of asset from the first account to the second.
func (s *UnitTestSuite) transfer(key string, asset *model.Asset, units int64) lt.RegisterPositionMovementInput {
	return lt.RegisterPositionMovementInput{
		IdempotencyKey: key,
		Kind:           model.PositionMovementTransfer,
		AssetID:        asset.ID,
		OccurredAt:     movementDay.AddDate(0, 0, 5),
		Legs:           []lt.PositionMovementLeg{movementLeg(s.financialAccount, -units), movementLeg(s.otherAccount, units)},
	}
}

func (s *UnitTestSuite) closeAccount(account *model.FinancialAccount) {
	s.T().Helper()

	closedAt := time.Now().UTC()
	account.ClosedAt = &closedAt
	_, err := s.DB.NewUpdate().Model(account).Column("closed_at").WherePK().Exec(s.T().Context())
	require.NoError(s.T(), err)
}

// closedAccount inserts a financial account that is already closed, so a case of
// a table can use one without closing an account the other cases share. Each
// caller names its own, since a name is unique within the institution.
func (s *UnitTestSuite) closedAccount(name string) *model.FinancialAccount {
	s.T().Helper()

	account := &model.FinancialAccount{InstitutionID: s.financialAccount.InstitutionID, Name: name}
	temporaltesting.Insert(s.T(), s.DB, account)
	s.closeAccount(account)
	return account
}

// Test_RegisterPositionMovement checks that each kind is written as one row per
// leg, under the key of the command and numbered in leg order, and that the
// units held afterwards are the sum of the legs.
func (s *UnitTestSuite) Test_RegisterPositionMovement() {
	tests := map[string]struct {
		// held is what the first account holds before the movement.
		held  int64
		input func(asset *model.Asset) lt.RegisterPositionMovementInput
		// wantFirst and wantSecond are what the first and the second account hold
		// afterwards.
		wantFirst, wantSecond int64
	}{
		"a split": {
			held: 10,
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("split-1", asset, 10)
				input.Kind, input.Description = model.PositionMovementSplit, nil
				input.Metadata = map[string]string{lt.SplitRatioMetadataKey: "2:1"}
				return input
			},
			wantFirst: 20,
		},
		"a transfer between accounts": {
			held:      10,
			input:     func(asset *model.Asset) lt.RegisterPositionMovementInput { return s.transfer("transfer-1", asset, 4) },
			wantFirst: 6, wantSecond: 4,
		},
		"a transfer with its cost basis": {
			held: 10,
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.transfer("transfer-2", asset, 4)
				input.Legs[0].CostBasisAmount, input.Legs[0].CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(-40)), ptr.Int64(s.eur.ID)
				input.Legs[1].CostBasisAmount, input.Legs[1].CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(40)), ptr.Int64(s.eur.ID)
				return input
			},
			wantFirst: 6, wantSecond: 4,
		},
		"a split of several accounts": {
			held: 10,
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.transfer("split-2", asset, 0)
				input.Kind = model.PositionMovementSplit
				input.Legs = []lt.PositionMovementLeg{movementLeg(s.financialAccount, 10), movementLeg(s.otherAccount, 3)}
				input.Metadata = map[string]string{lt.SplitRatioMetadataKey: "2:1"}
				return input
			},
			wantFirst: 20, wantSecond: 3,
		},
		"an adjustment": {
			held:      10,
			input:     func(asset *model.Asset) lt.RegisterPositionMovementInput { return s.adjustment("adjust-1", asset, -3) },
			wantFirst: 7,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			s.seedUnits(s.financialAccount, asset, tc.held, 1)
			input := tc.input(asset)

			activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)

			require.NoError(s.T(), err)
			var result lt.RegisterPositionMovementResult
			require.NoError(s.T(), activityResult.Get(&result))
			require.Len(s.T(), result.PositionMovementIDs, len(input.Legs))

			stored := s.movementsWithKey(input.IdempotencyKey)
			require.Len(s.T(), stored, len(input.Legs))
			for n, movement := range stored {
				require.Equal(s.T(), result.PositionMovementIDs[n], movement.ID, "ids are in leg order")
				require.Equal(s.T(), n+1, movement.LineNumber)
				require.Equal(s.T(), input.Kind, movement.Kind)
				require.Nil(s.T(), movement.JournalEntryID)
				require.Equal(s.T(), input.Legs[n].FinancialAccountID, movement.FinancialAccountID)
				require.True(s.T(), input.Legs[n].UnitsDelta.Equal(movement.UnitsDelta))
				require.True(s.T(), input.OccurredAt.Equal(movement.OccurredAt))
				require.Equal(s.T(), input.Description, movement.Description)
				require.Equal(s.T(), input.Metadata, movement.Metadata)
				require.Equal(s.T(), input.Legs[n].CostBasisCurrencyID, movement.CostBasisCurrencyID)
				if input.Legs[n].CostBasisAmount != nil {
					require.True(s.T(), input.Legs[n].CostBasisAmount.Equal(*movement.CostBasisAmount))
				}
			}

			require.True(s.T(), s.unitsHeld(s.financialAccount, asset).Equal(decimal.NewFromInt(tc.wantFirst)))
			require.True(s.T(), s.unitsHeld(s.otherAccount, asset).Equal(decimal.NewFromInt(tc.wantSecond)))
		})
	}
}

// Test_RegisterPositionMovement_Idempotent checks that a retry returns the rows
// of the first call and changes nothing, even when the retried request differs.
func (s *UnitTestSuite) Test_RegisterPositionMovement_Idempotent() {
	s.seedUnits(s.financialAccount, s.asset, 10, 1)
	input := s.transfer("transfer-retry", s.asset, 4)

	first, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.RegisterPositionMovementResult
	require.NoError(s.T(), first.Get(&created))
	rows := s.countMovements()

	second, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.RegisterPositionMovementResult
	require.NoError(s.T(), second.Get(&retried))

	require.Equal(s.T(), created.PositionMovementIDs, retried.PositionMovementIDs)
	require.Empty(s.T(), created.Warnings)
	require.Empty(s.T(), retried.Warnings)
	require.Equal(s.T(), rows, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.financialAccount, s.asset).Equal(decimal.NewFromInt(6)))
	require.True(s.T(), s.unitsHeld(s.otherAccount, s.asset).Equal(decimal.NewFromInt(4)))

	different := s.transfer("transfer-retry", s.asset, 9)
	third, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, different)
	require.NoError(s.T(), err)
	var unchanged lt.RegisterPositionMovementResult
	require.NoError(s.T(), third.Get(&unchanged))

	require.Equal(s.T(), created.PositionMovementIDs, unchanged.PositionMovementIDs)
	require.Equal(s.T(), rows, s.countMovements())
	require.True(s.T(), s.unitsHeld(s.otherAccount, s.asset).Equal(decimal.NewFromInt(4)))
}

// Test_RegisterPositionMovement_Negative checks that a movement is written
// whatever it leaves held, and that each position that is negative at some point
// in time is returned as a warning, ordered by financial account.
func (s *UnitTestSuite) Test_RegisterPositionMovement_Negative() {
	tests := map[string]struct {
		input func(asset *model.Asset) lt.RegisterPositionMovementInput
		// warnings are the expected warnings, in the order of the accounts.
		warnings []expectedWarning
		// wantFirst and wantSecond are what the first and the second account hold
		// afterwards.
		wantFirst, wantSecond int64
	}{
		"an adjustment below zero": {
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				return s.adjustment("below-zero", asset, -11)
			},
			warnings:  []expectedWarning{{s.financialAccount, negativeAt{-1, 5}}},
			wantFirst: -1,
		},
		"a transfer of more than the source holds": {
			input:     func(asset *model.Asset) lt.RegisterPositionMovementInput { return s.transfer("overdraw", asset, 11) },
			warnings:  []expectedWarning{{s.financialAccount, negativeAt{-1, 5}}},
			wantFirst: -1, wantSecond: 11,
		},
		"a split whose second leg goes below zero": {
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("half-written", asset, 10)
				input.Kind, input.Description = model.PositionMovementSplit, nil
				input.Metadata = map[string]string{lt.SplitRatioMetadataKey: "2:1"}
				input.Legs = []lt.PositionMovementLeg{movementLeg(s.financialAccount, 10), movementLeg(s.otherAccount, -1)}
				return input
			},
			warnings:  []expectedWarning{{s.otherAccount, negativeAt{-1, 5}}},
			wantFirst: 20, wantSecond: -1,
		},
		"a split that takes two accounts below zero": {
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("both-negative", asset, 0)
				input.Kind, input.Description = model.PositionMovementSplit, nil
				input.Metadata = map[string]string{lt.SplitRatioMetadataKey: "1:2"}
				input.Legs = []lt.PositionMovementLeg{movementLeg(s.otherAccount, -2), movementLeg(s.financialAccount, -12)}
				return input
			},
			warnings:  []expectedWarning{{s.financialAccount, negativeAt{-2, 5}}, {s.otherAccount, negativeAt{-2, 5}}},
			wantFirst: -2, wantSecond: -2,
		},
		"a movement dated before the holdings it needs": {
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("backdated", asset, -3)
				input.OccurredAt = movementDay
				return input
			},
			warnings:  []expectedWarning{{s.financialAccount, negativeAt{-3, 0}}},
			wantFirst: 7,
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			s.seedUnits(s.financialAccount, asset, 10, 1)
			input := tc.input(asset)
			rows := s.countMovements()

			activityResult, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)

			require.NoError(s.T(), err)
			var result lt.RegisterPositionMovementResult
			require.NoError(s.T(), activityResult.Get(&result))
			s.requireWarnings(result.Warnings, asset, tc.warnings...)
			require.Equal(s.T(), rows+len(input.Legs), s.countMovements())
			require.Len(s.T(), s.movementsWithKey(input.IdempotencyKey), len(input.Legs))
			require.True(s.T(), s.unitsHeld(s.financialAccount, asset).Equal(decimal.NewFromInt(tc.wantFirst)))
			require.True(s.T(), s.unitsHeld(s.otherAccount, asset).Equal(decimal.NewFromInt(tc.wantSecond)))
		})
	}
}

// Test_RegisterPositionMovement_IdempotentWarnings checks that a retry returns
// the warnings as they are now: the adjustment warns when it is made, and no
// longer once the units it needed have been backfilled.
func (s *UnitTestSuite) Test_RegisterPositionMovement_IdempotentWarnings() {
	input := s.adjustment("warning-retry", s.asset, -4)
	first, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.RegisterPositionMovementResult
	require.NoError(s.T(), first.Get(&created))
	s.requireWarnings(created.Warnings, s.asset, expectedWarning{s.financialAccount, negativeAt{-4, 5}})

	second, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.RegisterPositionMovementResult
	require.NoError(s.T(), second.Get(&retried))
	require.Equal(s.T(), created, retried)

	s.seedUnits(s.financialAccount, s.asset, 4, 1)

	third, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var afterBackfill lt.RegisterPositionMovementResult
	require.NoError(s.T(), third.Get(&afterBackfill))
	require.Equal(s.T(), created.PositionMovementIDs, afterBackfill.PositionMovementIDs)
	require.Empty(s.T(), afterBackfill.Warnings)
	require.Equal(s.T(), 2, s.countMovements())
}

// Test_RegisterPositionMovement_Rejected checks that a movement that cannot be
// written leaves no row behind, not even for a leg that is fine on its own.
func (s *UnitTestSuite) Test_RegisterPositionMovement_Rejected() {
	tests := map[string]struct {
		input func(asset *model.Asset) lt.RegisterPositionMovementInput
		// failure is what the error has to contain.
		failure string
	}{
		"a transfer to a closed account": {
			failure: "is closed",
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.transfer("to-closed", asset, 4)
				input.Legs[1] = movementLeg(s.closedAccount("Closed transfer destination"), 4)
				return input
			},
		},
		"an adjustment of a closed account": {
			failure: "is closed",
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("in-closed", asset, 1)
				input.Legs = []lt.PositionMovementLeg{movementLeg(s.closedAccount("Closed adjusted account"), 1)}
				return input
			},
		},
		"an asset that does not exist": {
			failure: "Asset",
			input: func(*model.Asset) lt.RegisterPositionMovementInput {
				return s.adjustment("no-asset", &model.Asset{ID: 999999}, 1)
			},
		},
		"a financial account that does not exist": {
			failure: "FinancialAccount",
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("no-account", asset, 1)
				input.Legs = []lt.PositionMovementLeg{movementLeg(&model.FinancialAccount{ID: 999999}, 1)}
				return input
			},
		},
		"a cost basis currency that does not exist": {
			failure: "Currency",
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("no-currency", asset, 1)
				input.Legs[0].CostBasisAmount, input.Legs[0].CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(10)), ptr.Int64(999999)
				return input
			},
		},
		"a trade": {
			failure: "trade",
			input: func(asset *model.Asset) lt.RegisterPositionMovementInput {
				input := s.adjustment("as-trade", asset, 1)
				input.Kind = model.PositionMovementTrade
				return input
			},
		},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			asset := &model.Asset{Name: name, Kind: model.AssetKindStock}
			temporaltesting.Insert(s.T(), s.DB, asset)
			s.seedUnits(s.financialAccount, asset, 10, 1)
			input := tc.input(asset)
			rows := s.countMovements()

			_, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)

			require.ErrorContains(s.T(), err, tc.failure)
			require.Equal(s.T(), rows, s.countMovements(), "a rejected movement must leave nothing behind")
		})
	}
}

// Test_RegisterPositionMovement_ClosedAccountRetry checks that a retry of a
// movement made while the account was open still returns it after the account
// is closed.
func (s *UnitTestSuite) Test_RegisterPositionMovement_ClosedAccountRetry() {
	input := s.adjustment("before-close", s.asset, 10)
	first, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var created lt.RegisterPositionMovementResult
	require.NoError(s.T(), first.Get(&created))

	s.closeAccount(s.financialAccount)

	retry, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, input)
	require.NoError(s.T(), err)
	var retried lt.RegisterPositionMovementResult
	require.NoError(s.T(), retry.Get(&retried))
	require.Equal(s.T(), created.PositionMovementIDs, retried.PositionMovementIDs)
	require.Equal(s.T(), 1, s.countMovements())
}

// holdAccount opens a transaction that has locked the financial account, as a
// concurrent writer of it would, and inserts movements in it. The transaction
// stays open until the returned function commits it.
func (s *UnitTestSuite) holdAccount(account *model.FinancialAccount, movements ...*model.PositionMovement) (commit func()) {
	s.T().Helper()

	writer, err := s.DB.BeginTx(s.T().Context(), nil)
	require.NoError(s.T(), err)
	s.T().Cleanup(func() { _ = writer.Rollback() })

	_, err = writer.ExecContext(s.T().Context(), "SELECT id FROM account.financial_account WHERE id = ? FOR NO KEY UPDATE", account.ID)
	require.NoError(s.T(), err)
	for _, movement := range movements {
		_, err = writer.NewInsert().Model(temporaltesting.WithMovementDefaults(movement)).Exec(s.T().Context())
		require.NoError(s.T(), err)
	}
	return func() { require.NoError(s.T(), writer.Commit()) }
}

type movementOutcome struct {
	result *lt.RegisterPositionMovementResult
	err    error
}

// registerInBackground runs the activity directly, outside the test environment,
// so that the test can look at it while it is blocked.
func (s *UnitTestSuite) registerInBackground(input lt.RegisterPositionMovementInput) <-chan movementOutcome {
	done := make(chan movementOutcome, 1)
	go func() {
		result, err := s.pa.RegisterPositionMovement(s.T().Context(), input)
		done <- movementOutcome{result, err}
	}()
	return done
}

func (s *UnitTestSuite) requireStillWaiting(done <-chan movementOutcome, what string) {
	s.T().Helper()

	select {
	case <-done:
		s.T().Fatalf("%s finished without waiting for the transaction holding the account", what)
	case <-time.After(500 * time.Millisecond):
	}
}

// Test_RegisterPositionMovement_WaitsForConcurrentWriter checks that a movement
// waits for a transaction that has locked the financial account, and then
// counts what it wrote. A concurrent adjustment of -4 out of 6 units is held
// open here, so without the lock the second one would see 6 units and report no
// warning, although the position is left at -2.
func (s *UnitTestSuite) Test_RegisterPositionMovement_WaitsForConcurrentWriter() {
	s.seedUnits(s.financialAccount, s.asset, 6, 1)
	commit := s.holdAccount(s.financialAccount, &model.PositionMovement{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            s.asset.ID,
		UnitsDelta:         decimal.NewFromInt(-4),
		OccurredAt:         movementDay.AddDate(0, 0, 2),
	})

	done := s.registerInBackground(s.adjustment("racing-adjustment", s.asset, -4))
	s.requireStillWaiting(done, "the movement")

	commit()
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	s.requireWarnings(outcome.result.Warnings, s.asset, expectedWarning{s.financialAccount, negativeAt{-2, 5}})
}

// Test_RegisterPositionMovement_RetryWaitsForConcurrentWriter checks that a
// retry that arrives while the first call is still in its transaction waits for
// it and returns its rows, instead of missing them and colliding on the key.
func (s *UnitTestSuite) Test_RegisterPositionMovement_RetryWaitsForConcurrentWriter() {
	key := "racing-key"
	commit := s.holdAccount(s.financialAccount, &model.PositionMovement{
		FinancialAccountID: s.financialAccount.ID,
		AssetID:            s.asset.ID,
		UnitsDelta:         decimal.NewFromInt(5),
		OccurredAt:         movementDay,
		IdempotencyKey:     &key,
		LineNumber:         1,
	})

	done := s.registerInBackground(s.adjustment(key, s.asset, 5))
	s.requireStillWaiting(done, "the retry")

	commit()
	outcome := <-done
	require.NoError(s.T(), outcome.err)
	require.Equal(s.T(), 1, s.countMovements())
	require.Equal(s.T(), []int64{s.movementsWithKey(key)[0].ID}, outcome.result.PositionMovementIDs)
}

// Test_RegisterPositionMovement_KeyOfReversal checks that a key that belongs to
// a reversal is not taken for the retry of a registration.
func (s *UnitTestSuite) Test_RegisterPositionMovement_KeyOfReversal() {
	originals := s.seedTransfer("transfer-1", s.asset)
	for n, original := range originals {
		temporaltesting.Insert(s.T(), s.DB, &model.PositionMovement{
			Kind: original.Kind, FinancialAccountID: original.FinancialAccountID, AssetID: s.asset.ID,
			UnitsDelta: original.UnitsDelta.Neg(), OccurredAt: movementDay.AddDate(0, 0, 6),
			IdempotencyKey: ptr.NewPtr("reversal-1"), LineNumber: n + 1, ReversalOfMovementID: &original.ID,
		})
	}
	rows := s.countMovements()

	_, err := s.Env.ExecuteActivity(s.pa.RegisterPositionMovement, s.adjustment("reversal-1", s.asset, 1))

	require.ErrorContains(s.T(), err, "belongs to a reversal")
	require.Equal(s.T(), rows, s.countMovements())
}
