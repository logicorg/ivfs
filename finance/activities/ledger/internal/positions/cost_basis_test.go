package positions

import (
	"testing"

	"github.com/logicorg/ivfs/pkg/model"
	"github.com/logicorg/ivfs/pkg/ptr"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

const (
	eur int64 = 1
	usd int64 = 2
)

// step is a movement of the asset held in account 1, unless asset says
// otherwise. The steps of a case are replayed in the order given, and the n-th
// has id n+1.
type step struct {
	asset    int64
	units    string
	basis    string
	currency int64
	kind     model.PositionMovementKind
	// reverses is the id of the movement this one reverses.
	reverses int64
}

type replayed struct {
	units   string
	amounts map[int64]string
	unknown bool
	// average is the average cost in EUR, or empty when there is none.
	average string
}

func Test_ReplayCostBasis(t *testing.T) {
	tests := map[string]struct {
		steps []step
		// want is the basis by asset.
		want map[int64]replayed
	}{
		"buys add their basis": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "5", basis: "80", currency: eur}},
			want:  map[int64]replayed{1: {units: "15", amounts: map[int64]string{eur: "180"}, average: "12"}},
		},
		"a sale without a basis removes the average cost": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "10", basis: "200", currency: eur}, {units: "-5"}},
			want:  map[int64]replayed{1: {units: "15", amounts: map[int64]string{eur: "225"}, average: "15"}},
		},
		"the removal is rounded to the basis column's scale": {
			steps: []step{{units: "3", basis: "100", currency: eur}, {units: "-1"}},
			want:  map[int64]replayed{1: {units: "2", amounts: map[int64]string{eur: "66.666666666666666667"}, average: "33.333333333333333334"}},
		},
		"a sale with a basis removes the basis it states": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "-4", basis: "-50", currency: eur}},
			want:  map[int64]replayed{1: {units: "6", amounts: map[int64]string{eur: "50"}, average: "8.333333333333333333"}},
		},
		"a buy without a basis makes it unknown": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "5"}},
			want:  map[int64]replayed{1: {units: "15", amounts: map[int64]string{eur: "100"}, unknown: true}},
		},
		"a sale while it is unknown keeps it unknown": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "5"}, {units: "-3"}},
			want:  map[int64]replayed{1: {units: "12", amounts: map[int64]string{eur: "100"}, unknown: true}},
		},
		"selling every unit makes it a known zero": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "5"}, {units: "-15"}, {units: "4", basis: "40", currency: eur}},
			want:  map[int64]replayed{1: {units: "4", amounts: map[int64]string{eur: "40"}, average: "10"}},
		},
		"a sale of a basis in two currencies is unknown": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "10", basis: "50", currency: usd}, {units: "-5"}},
			want:  map[int64]replayed{1: {units: "15", amounts: map[int64]string{eur: "100", usd: "50"}, unknown: true}},
		},
		"selling every unit of a basis in two currencies is a known zero": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "10", basis: "50", currency: usd}, {units: "-20"}},
			want:  map[int64]replayed{1: {units: "0", amounts: map[int64]string{}}},
		},
		"a sale of units not held is unknown": {
			steps: []step{{units: "-5"}, {units: "10", basis: "100", currency: eur}},
			want:  map[int64]replayed{1: {units: "5", amounts: map[int64]string{eur: "100"}, unknown: true}},
		},
		"a sale of a holding that cost nothing stays known": {
			steps: []step{{units: "10", basis: "0", currency: eur}, {units: "-4"}},
			want:  map[int64]replayed{1: {units: "6", amounts: map[int64]string{}}},
		},
		"a split spreads the basis over more units": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "10", kind: model.PositionMovementSplit}},
			want:  map[int64]replayed{1: {units: "20", amounts: map[int64]string{eur: "100"}, average: "5"}},
		},
		"a reverse split keeps the basis": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "-5", kind: model.PositionMovementSplit}},
			want:  map[int64]replayed{1: {units: "5", amounts: map[int64]string{eur: "100"}, average: "20"}},
		},
		"a reversed sale and its reversal are skipped": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "-4"}, {units: "4", reverses: 2}},
			want:  map[int64]replayed{1: {units: "10", amounts: map[int64]string{eur: "100"}, average: "10"}},
		},
		"a reversed buy and its reversal are skipped": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "5"}, {units: "-5", reverses: 2}},
			want:  map[int64]replayed{1: {units: "10", amounts: map[int64]string{eur: "100"}, average: "10"}},
		},
		"a reversal without its original is replayed": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {units: "4", reverses: 99}},
			want:  map[int64]replayed{1: {units: "14", amounts: map[int64]string{eur: "100"}, unknown: true}},
		},
		"positions are replayed apart": {
			steps: []step{{units: "10", basis: "100", currency: eur}, {asset: 2, units: "5"}, {units: "-5"}},
			want: map[int64]replayed{
				1: {units: "5", amounts: map[int64]string{eur: "50"}, average: "10"},
				2: {units: "5", amounts: map[int64]string{}, unknown: true},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			movements := make([]model.PositionMovement, len(tc.steps))
			for n, s := range tc.steps {
				m := model.PositionMovement{
					ID: int64(n + 1), FinancialAccountID: 1, AssetID: 1,
					Kind: model.PositionMovementAdjustment, UnitsDelta: decimal.RequireFromString(s.units),
				}
				if s.asset != 0 {
					m.AssetID = s.asset
				}
				if s.kind != "" {
					m.Kind = s.kind
				}
				if s.basis != "" {
					m.CostBasisAmount, m.CostBasisCurrencyID = ptr.NewPtr(decimal.RequireFromString(s.basis)), ptr.NewPtr(s.currency)
				}
				if s.reverses != 0 {
					m.ReversalOfMovementID = ptr.NewPtr(s.reverses)
				}
				movements[n] = m
			}

			bases := ReplayCostBasis(movements)

			got := make(map[int64]replayed, len(bases))
			for pair, basis := range bases {
				amounts := make(map[int64]string, len(basis.Amounts))
				for currencyID, amount := range basis.Amounts {
					amounts[currencyID] = amount.String()
				}
				r := replayed{units: basis.Units.String(), amounts: amounts, unknown: basis.Unknown}
				if currencyID, average, ok := basis.Average(); ok {
					require.Equal(t, eur, currencyID)
					r.average = average.String()
				}
				got[pair.AssetID] = r
			}
			require.Equal(t, tc.want, got)
		})
	}
}
