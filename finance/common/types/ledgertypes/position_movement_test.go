package ledgertypes

import (
	"testing"
	"time"

	internalerrors "github.com/logicorg/ivfs/internal/pkg/errors"
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestRegisterPositionMovementInput_Validate(t *testing.T) {
	leg := func(account, units int64) PositionMovementLeg {
		return PositionMovementLeg{FinancialAccountID: account, UnitsDelta: decimal.NewFromInt(units)}
	}
	withBasis := func(l PositionMovementLeg, amount, currency int64) PositionMovementLeg {
		l.CostBasisAmount, l.CostBasisCurrencyID = ptr.NewPtr(decimal.NewFromInt(amount)), ptr.Int64(currency)
		return l
	}

	// valid is an adjustment, which has the fewest rules; the cases turn it into
	// the other kinds.
	valid := func() RegisterPositionMovementInput {
		return RegisterPositionMovementInput{
			IdempotencyKey: "adjust-1",
			Kind:           model.PositionMovementAdjustment,
			AssetID:        1,
			OccurredAt:     time.Now(),
			Description:    ptr.NewPtr("broker correction"),
			Legs:           []PositionMovementLeg{leg(1, 5)},
		}
	}
	asSplit := func(i *RegisterPositionMovementInput) {
		i.Kind, i.Description = model.PositionMovementSplit, nil
		i.Metadata = map[string]string{SplitRatioMetadataKey: "2:1"}
	}
	asTransfer := func(i *RegisterPositionMovementInput) {
		i.Kind, i.Description = model.PositionMovementTransfer, nil
		i.Legs = []PositionMovementLeg{leg(1, -5), leg(2, 5)}
	}

	const (
		accepted   = ""
		constraint = "cue"
		rule       = "rule"
		missing    = "missing"
	)

	tests := map[string]struct {
		mutate func(*RegisterPositionMovementInput)
		// failure is how the input is rejected: by a CUE constraint, which is a
		// ValidationError, by a rule CUE cannot express, which is an
		// ErrInvalidParameter, or because a value is absent.
		failure string
	}{
		"an adjustment": {mutate: func(*RegisterPositionMovementInput) {}},
		"an adjustment of several accounts": {mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{leg(1, 5), leg(2, -3)}
		}},
		"an adjustment with a cost basis": {mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{withBasis(leg(1, 5), 50, 1)}
		}},
		"a split": {mutate: asSplit},
		"a split of several accounts": {mutate: func(i *RegisterPositionMovementInput) {
			asSplit(i)
			i.Legs = []PositionMovementLeg{leg(1, 5), leg(2, 15)}
		}},
		"a split with a zero cost basis": {mutate: func(i *RegisterPositionMovementInput) {
			asSplit(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, 5), 0, 1)}
		}},
		"a transfer": {mutate: asTransfer},
		"a transfer with a cost basis": {mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, -5), -50, 1), withBasis(leg(2, 5), 50, 1)}
		}},
		"no idempotency key": {failure: constraint, mutate: func(i *RegisterPositionMovementInput) {
			i.IdempotencyKey = ""
		}},
		"no asset": {failure: constraint, mutate: func(i *RegisterPositionMovementInput) {
			i.AssetID = 0
		}},
		"no date": {failure: missing, mutate: func(i *RegisterPositionMovementInput) {
			i.OccurredAt = time.Time{}
		}},
		"a trade": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Kind = model.PositionMovementTrade
		}},
		"a vest": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Kind = model.PositionMovementVest
		}},
		"an unknown kind": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Kind = "gift"
		}},
		"no legs": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = nil
		}},
		"a leg without an account": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{leg(0, 5)}
		}},
		"a leg without units": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{leg(1, 0)}
		}},
		"two legs on one account": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{leg(1, 5), leg(1, -2)}
		}},
		"a cost basis without a currency": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{{FinancialAccountID: 1, UnitsDelta: decimal.NewFromInt(5), CostBasisAmount: ptr.NewPtr(decimal.NewFromInt(50))}}
		}},
		"a cost basis currency without an amount": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{{FinancialAccountID: 1, UnitsDelta: decimal.NewFromInt(5), CostBasisCurrencyID: ptr.Int64(1)}}
		}},
		"a cost basis that is not an id": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{withBasis(leg(1, 5), 50, 0)}
		}},
		"a cost basis against the sign of the units": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			i.Legs = []PositionMovementLeg{withBasis(leg(1, 5), -50, 1)}
		}},
		"an adjustment without a description": {failure: missing, mutate: func(i *RegisterPositionMovementInput) {
			i.Description = nil
		}},
		"an adjustment with a blank description": {failure: missing, mutate: func(i *RegisterPositionMovementInput) {
			i.Description = ptr.NewPtr("  ")
		}},
		"a split without a ratio": {failure: missing, mutate: func(i *RegisterPositionMovementInput) {
			asSplit(i)
			i.Metadata = nil
		}},
		"a split with a blank ratio": {failure: missing, mutate: func(i *RegisterPositionMovementInput) {
			asSplit(i)
			i.Metadata[SplitRatioMetadataKey] = " "
		}},
		"a split with a cost basis": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asSplit(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, 5), 50, 1)}
		}},
		"a transfer of one leg": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = i.Legs[:1]
		}},
		"a transfer of three legs": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{leg(1, -5), leg(2, 3), leg(3, 2)}
		}},
		"a transfer between one account": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{leg(1, -5), leg(1, 5)}
		}},
		"a transfer of unequal units": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{leg(1, -5), leg(2, 4)}
		}},
		"a transfer with the cost basis of the first leg only": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, -5), -50, 1), leg(2, 5)}
		}},
		"a transfer with the cost basis of the second leg only": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{leg(1, -5), withBasis(leg(2, 5), 50, 1)}
		}},
		"a transfer of unequal cost basis": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, -5), -50, 1), withBasis(leg(2, 5), 40, 1)}
		}},
		"a transfer with the cost basis in two currencies": {failure: rule, mutate: func(i *RegisterPositionMovementInput) {
			asTransfer(i)
			i.Legs = []PositionMovementLeg{withBasis(leg(1, -5), -50, 1), withBasis(leg(2, 5), 50, 2)}
		}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			input := valid()
			tc.mutate(&input)

			err := input.Validate()

			switch tc.failure {
			case accepted:
				require.NoError(t, err)
			case constraint:
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
			case rule:
				require.ErrorAs(t, err, &internalerrors.ErrInvalidParameter{})
			case missing:
				require.ErrorAs(t, err, &internalerrors.ErrMissingRequiredParameter{})
			}
		})
	}
}

// TestRegisterPositionMovementInput_ToMovements checks that the legs become
// movements that share the key and are numbered from 1 in leg order.
func TestRegisterPositionMovementInput_ToMovements(t *testing.T) {
	input := RegisterPositionMovementInput{
		IdempotencyKey: "transfer-1",
		Kind:           model.PositionMovementTransfer,
		AssetID:        7,
		OccurredAt:     time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),
		ExternalID:     ptr.NewPtr("broker-ref"),
		Legs: []PositionMovementLeg{
			{FinancialAccountID: 1, UnitsDelta: decimal.NewFromInt(-5)},
			{FinancialAccountID: 2, UnitsDelta: decimal.NewFromInt(5)},
		},
	}

	movements := input.ToMovements()

	require.Len(t, movements, 2)
	require.Equal(t, []int64{1, 2}, input.FinancialAccountIDs())
	for n, movement := range movements {
		require.Equal(t, model.PositionMovementTransfer, movement.Kind)
		require.Equal(t, int64(7), movement.AssetID)
		require.Equal(t, "transfer-1", *movement.IdempotencyKey)
		require.Equal(t, n+1, movement.LineNumber)
		require.Equal(t, input.OccurredAt, movement.OccurredAt)
		require.Equal(t, "broker-ref", *movement.ExternalID)
		require.Nil(t, movement.JournalEntryID)
	}
	require.True(t, movements[0].UnitsDelta.Equal(decimal.NewFromInt(-5)))
	require.True(t, movements[1].UnitsDelta.Equal(decimal.NewFromInt(5)))
}

func TestReversePositionMovementInput_Validate(t *testing.T) {
	tests := map[string]struct {
		input   ReversePositionMovementInput
		wantErr bool
	}{
		"a movement and a key": {input: ReversePositionMovementInput{MovementID: 1, IdempotencyKey: "reversal-1"}},
		"a date":               {input: ReversePositionMovementInput{MovementID: 1, IdempotencyKey: "reversal-1", OccurredAt: time.Now()}},
		"no movement":          {input: ReversePositionMovementInput{IdempotencyKey: "reversal-1"}, wantErr: true},
		"a negative movement":  {input: ReversePositionMovementInput{MovementID: -1, IdempotencyKey: "reversal-1"}, wantErr: true},
		"no idempotency key":   {input: ReversePositionMovementInput{MovementID: 1}, wantErr: true},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.input.Validate()

			if tc.wantErr {
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestListPositionMovementsParams_Validate(t *testing.T) {
	day := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)

	const (
		accepted   = ""
		constraint = "cue"
		rule       = "rule"
	)

	tests := map[string]struct {
		params ListPositionMovementsParams
		// failure is how the parameters are rejected: by a CUE constraint, which is
		// a ValidationError, or by a rule CUE cannot express, which is an
		// ErrInvalidParameter.
		failure string
	}{
		"no filter":                          {params: ListPositionMovementsParams{}},
		"an account and an asset":            {params: ListPositionMovementsParams{FinancialAccountID: 1, AssetID: 2}},
		"kinds":                              {params: ListPositionMovementsParams{Kinds: []model.PositionMovementKind{model.PositionMovementSplit, model.PositionMovementTrade}}},
		"a range":                            {params: ListPositionMovementsParams{From: day, To: day.Add(time.Hour)}},
		"a range with no start":              {params: ListPositionMovementsParams{To: day}},
		"a range with no end":                {params: ListPositionMovementsParams{From: day}},
		"the largest page":                   {params: ListPositionMovementsParams{Limit: MaxPositionMovementsLimit}},
		"a cursor":                           {params: ListPositionMovementsParams{After: &PositionMovementCursor{OccurredAt: day, ID: 3}}},
		"a negative account":                 {failure: constraint, params: ListPositionMovementsParams{FinancialAccountID: -1}},
		"a negative asset":                   {failure: constraint, params: ListPositionMovementsParams{AssetID: -1}},
		"a negative limit":                   {failure: constraint, params: ListPositionMovementsParams{Limit: -1}},
		"a page that is too large":           {failure: rule, params: ListPositionMovementsParams{Limit: MaxPositionMovementsLimit + 1}},
		"a range that ends where it starts":  {failure: rule, params: ListPositionMovementsParams{From: day, To: day}},
		"a range that ends before it starts": {failure: rule, params: ListPositionMovementsParams{From: day, To: day.Add(-time.Hour)}},
		"an unknown kind":                    {failure: rule, params: ListPositionMovementsParams{Kinds: []model.PositionMovementKind{"gift"}}},
		"a cursor without an id":             {failure: rule, params: ListPositionMovementsParams{After: &PositionMovementCursor{OccurredAt: day}}},
		"a cursor without a date":            {failure: rule, params: ListPositionMovementsParams{After: &PositionMovementCursor{ID: 3}}},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := tc.params.Validate()

			switch tc.failure {
			case accepted:
				require.NoError(t, err)
			case constraint:
				require.ErrorAs(t, err, &internalerrors.ValidationError{})
			case rule:
				require.ErrorAs(t, err, &internalerrors.ErrInvalidParameter{})
			}
		})
	}
}

func TestListPositionMovementsParams_PageSize(t *testing.T) {
	require.Equal(t, DefaultPositionMovementsLimit, (&ListPositionMovementsParams{}).PageSize())
	require.Equal(t, 7, (&ListPositionMovementsParams{Limit: 7}).PageSize())
}
