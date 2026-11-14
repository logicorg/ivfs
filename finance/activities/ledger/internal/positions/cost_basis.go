package positions

import (
	"github.com/logicorg/ivfs/pkg/model"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

// basisPlaces is the scale of the basis column, to which an average-cost
// removal is rounded, so a replayed basis has no more digits than a stored one.
const basisPlaces = 18

// CostBasis is the cost basis of one asset held in one financial account,
// replayed from its movements (ADR 0009).
type CostBasis struct {
	// Units is the units held after the movements replayed.
	Units decimal.Decimal
	// Amounts is the basis in each currency it is in, by currency id, without
	// the zero ones. While Unknown, it is only the part the movements stated,
	// not what the units cost.
	Amounts map[int64]decimal.Decimal
	Unknown bool
}

// Average returns the cost of one unit and its currency. There is none while
// the units are zero or negative, the basis is unknown, or it is in more than
// one currency, since amounts in different currencies cannot be added.
func (b *CostBasis) Average() (currencyID int64, amount decimal.Decimal, ok bool) {
	if b.Unknown || !b.Units.IsPositive() || len(b.Amounts) != 1 {
		return 0, decimal.Zero, false
	}
	for currencyID, amount := range b.Amounts {
		return currencyID, amount.DivRound(b.Units, basisPlaces), true
	}
	return 0, decimal.Zero, false
}

func (b *CostBasis) apply(m model.PositionMovement) {
	held := b.Units
	b.Units = held.Add(m.UnitsDelta)

	switch {
	case m.CostBasisAmount != nil:
		b.add(*m.CostBasisCurrencyID, *m.CostBasisAmount)
	case m.Kind == model.PositionMovementSplit:
		// A split changes how many units the same cost is spread over.
	case m.UnitsDelta.IsPositive():
		b.Unknown = true
	case b.Units.IsZero():
		// Selling everything removes the basis, whatever it was, so the holding
		// starts again from a known zero.
		b.Amounts = map[int64]decimal.Decimal{}
		b.Unknown = false
	case b.Unknown || !held.IsPositive() || len(b.Amounts) > 1:
		b.Unknown = true
	default:
		// The units left keep their share of the basis, which removes the
		// average cost of the ones that went.
		for currencyID, amount := range b.Amounts {
			b.Amounts[currencyID] = amount.Mul(b.Units).DivRound(held, basisPlaces)
		}
	}
}

func (b *CostBasis) add(currencyID int64, amount decimal.Decimal) {
	total := b.Amounts[currencyID].Add(amount)
	if total.IsZero() {
		delete(b.Amounts, currencyID)
		return
	}
	b.Amounts[currencyID] = total
}

// CostBasisMovementsQuery selects the effective movements UnitsQuery counts,
// in the order their basis is replayed: by position, then when they occurred,
// then when they were recorded, then by id.
func CostBasisMovementsQuery(db bun.IDB, filter Filter) *bun.SelectQuery {
	query := EffectiveMovementsQuery(db).OrderExpr("pm.financial_account_id, pm.asset_id, pm.occurred_at, pm.created_at, pm.id")
	if filter.FinancialAccountID > 0 {
		query = query.Where("pm.financial_account_id = ?", filter.FinancialAccountID)
	}
	if filter.AssetID > 0 {
		query = query.Where("pm.asset_id = ?", filter.AssetID)
	}
	if !filter.At.IsZero() {
		query = query.Where("pm.occurred_at <= ?", filter.At)
	}
	return query
}

// ReplayCostBasis replays the movements CostBasisMovementsQuery selects, in
// its order, and returns the basis of each position.
//
// A movement without a basis removes the average cost when it decreases the
// units, and makes the basis unknown when it increases them, except a split.
func ReplayCostBasis(movements []model.PositionMovement) map[Pair]*CostBasis {
	bases := make(map[Pair]*CostBasis)
	for _, m := range movements {
		pair := Pair{FinancialAccountID: m.FinancialAccountID, AssetID: m.AssetID}
		basis, ok := bases[pair]
		if !ok {
			basis = &CostBasis{Amounts: map[int64]decimal.Decimal{}}
			bases[pair] = basis
		}
		basis.apply(m)
	}
	return bases
}
