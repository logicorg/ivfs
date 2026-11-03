package sqllib_test

import (
	"time"

	"github.com/logicorg/ivfs/internal/pkg/temporaltesting"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/model/testhelpers"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/logicorg/ivfs/pkg/sqllib"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

// Test_PositionMovementConstraints checks the rules of a position movement that
// the database enforces, running the inserts directly, as anything other than
// the activities would.
func (s *UnitTestSuite) Test_PositionMovementConstraints() {
	eur := testhelpers.Euro()
	temporaltesting.Insert(s.T(), s.DB, eur)
	institution := &model.Institution{Name: "Broker", Kind: "broker"}
	temporaltesting.Insert(s.T(), s.DB, institution)
	account := &model.FinancialAccount{InstitutionID: institution.ID, Name: "Broker account"}
	temporaltesting.Insert(s.T(), s.DB, account)
	asset := &model.Asset{Name: "AAPL", Kind: model.AssetKindStock}
	temporaltesting.Insert(s.T(), s.DB, asset)
	entry := &model.JournalEntry{OccurredAt: time.Now().UTC(), Status: model.JournalEntryDraft}
	temporaltesting.Insert(s.T(), s.DB, entry)

	// adjustment is a movement with no journal entry, identified by its own key.
	adjustment := func(key string) *model.PositionMovement {
		return &model.PositionMovement{
			Kind:               model.PositionMovementAdjustment,
			FinancialAccountID: account.ID,
			AssetID:            asset.ID,
			UnitsDelta:         decimal.NewFromInt(5),
			OccurredAt:         time.Now().UTC(),
			IdempotencyKey:     ptr.NewPtr(key),
			LineNumber:         1,
		}
	}
	trade := func() *model.PositionMovement {
		return &model.PositionMovement{
			Kind:               model.PositionMovementTrade,
			JournalEntryID:     &entry.ID,
			FinancialAccountID: account.ID,
			AssetID:            asset.ID,
			UnitsDelta:         decimal.NewFromInt(5),
			OccurredAt:         entry.OccurredAt,
		}
	}

	// existing inserts an adjustment first and returns its id, for the cases that
	// conflict with a row that is already there.
	existing := func(key string) int64 {
		original := adjustment(key)
		temporaltesting.Insert(s.T(), s.DB, original)
		return original.ID
	}

	tests := map[string]struct {
		movement func() *model.PositionMovement
		// constraint is the constraint the insert has to violate, or empty when it
		// has to be accepted.
		constraint string
	}{
		"an adjustment with its own key": {movement: func() *model.PositionMovement { return adjustment("adjust") }},
		"a trade with an entry":          {movement: trade},
		"a cost basis with its currency": {movement: func() *model.PositionMovement {
			m := trade()
			m.CostBasisAmount, m.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(100)), ptr.Int64(eur.ID)
			return m
		}},
		"an unknown kind": {constraint: "position_movement_kind_check", movement: func() *model.PositionMovement {
			m := adjustment("unknown-kind")
			m.Kind = "gift"
			return m
		}},
		"no units": {constraint: "position_movement_units_nonzero_check", movement: func() *model.PositionMovement {
			m := adjustment("no-units")
			m.UnitsDelta = decimal.Zero
			return m
		}},
		"a trade without an entry": {constraint: "position_movement_entry_kind_check", movement: func() *model.PositionMovement {
			m := adjustment("trade-without-entry")
			m.Kind = model.PositionMovementTrade
			return m
		}},
		"an adjustment with an entry": {constraint: "position_movement_entry_kind_check", movement: func() *model.PositionMovement {
			m := trade()
			m.Kind = model.PositionMovementAdjustment
			return m
		}},
		"a movement without an entry or a key": {constraint: "position_movement_entryless_key_check", movement: func() *model.PositionMovement {
			m := adjustment("")
			m.IdempotencyKey, m.LineNumber = nil, 0
			return m
		}},
		"a key without a line": {constraint: "position_movement_key_line_check", movement: func() *model.PositionMovement {
			m := adjustment("key-without-line")
			m.LineNumber = 0
			return m
		}},
		"a line without a key": {constraint: "position_movement_key_line_check", movement: func() *model.PositionMovement {
			m := trade()
			m.LineNumber = 1
			return m
		}},
		"a cost basis without a currency": {constraint: "position_movement_cost_basis_currency_check", movement: func() *model.PositionMovement {
			m := trade()
			m.CostBasisAmount = ptr.NewPtr(decimal.NewFromInt(100))
			return m
		}},
		"a cost basis currency without an amount": {constraint: "position_movement_cost_basis_currency_check", movement: func() *model.PositionMovement {
			m := trade()
			m.CostBasisCurrencyID = ptr.Int64(eur.ID)
			return m
		}},
		"the same key and line twice": {constraint: "position_movement_idempotency_line_unique", movement: func() *model.PositionMovement {
			existing("twice")
			return adjustment("twice")
		}},
		"a movement reversed twice": {constraint: "position_movement_reversal_unique", movement: func() *model.PositionMovement {
			original := existing("reversed")
			first := adjustment("first-reversal")
			first.UnitsDelta, first.ReversalOfMovementID = decimal.NewFromInt(-5), &original
			temporaltesting.Insert(s.T(), s.DB, first)

			second := adjustment("second-reversal")
			second.UnitsDelta, second.ReversalOfMovementID = decimal.NewFromInt(-5), &original
			return second
		}},
		"a reversal of a movement that does not exist": {constraint: "position_movement_reversal_of_movement_id_fkey", movement: func() *model.PositionMovement {
			m := adjustment("dangling-reversal")
			m.ReversalOfMovementID = ptr.Int64(999999)
			return m
		}},
		"a cost basis in a currency that does not exist": {constraint: "position_movement_cost_basis_currency_id_fkey", movement: func() *model.PositionMovement {
			m := trade()
			m.CostBasisAmount, m.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(100)), ptr.Int64(999999)
			return m
		}},
	}

	for name, tc := range tests {
		s.Run(name, func() {
			movement := tc.movement()

			_, err := s.DB.NewInsert().Model(movement).Exec(s.T().Context())

			if tc.constraint == "" {
				require.NoError(s.T(), err)
				return
			}
			require.ErrorContains(s.T(), err, tc.constraint)
		})
	}
}

// Test_PositionMovementRulesAreInstalledOnce checks that installing the schema
// again, as every start of the worker does, neither fails nor duplicates the
// rules.
func (s *UnitTestSuite) Test_PositionMovementRulesAreInstalledOnce() {
	require.NoError(s.T(), sqllib.InitDatabase(s.T().Context(), s.DB))

	var count int
	err := s.DB.NewRaw(`SELECT COUNT(*) FROM pg_constraint WHERE conrelid = 'asset.position_movement'::regclass AND contype IN ('c', 'u')`).Scan(s.T().Context(), &count)
	require.NoError(s.T(), err)
	require.Equal(s.T(), 8, count, "six checks and two unique constraints")
}
