package ledgertypes

import (
	"time"

	"github.com/shopspring/decimal"
)

// FinancialAccountBalancesParams is the input struct to retrieve a FinancialAccount balance at a given date.
type FinancialAccountBalancesParams struct {
	FinancialAccountID int64     `json:"financialAccountId"`
	At                 time.Time `json:"at,omitzero" jsonschema:"count only entries that occurred at or before this RFC 3339 instant, such as 2026-01-31T23:59:59Z; now if omitted"`
}

// FinancialAccountBalancesResult holds the result of an operation to get the FinancialAccount balance.
type FinancialAccountBalancesResult struct {
	FinancialAccountID int64                        `json:"financialAccountId"`
	Balances           []LedgerAccountBalanceResult `json:"balances"`
}

type PositionsAtParams struct {
	FinancialAccountID int64     `json:"financialAccountId,omitempty"`
	AssetID            int64     `json:"assetId,omitempty"`
	At                 time.Time `json:"at,omitempty"`
}

// PositionResult is the units of one asset held in one financial account, and
// what they cost.
type PositionResult struct {
	FinancialAccountID int64           `json:"financialAccountId"`
	AssetID            int64           `json:"assetId"`
	Units              decimal.Decimal `json:"units"`
	// CostBasis is the cost of the units held in each currency it was paid
	// in, by currency id, replayed from the movements: a sale removes the
	// average cost of the units it sold unless it states its own basis
	// (ADR 0009). An asset can be bought in several currencies, and the
	// amounts are not converted, so they cannot be added together.
	CostBasis []CostBasisAmount `json:"costBasis"`
	// UnknownCostBasis is true when a movement added units without a basis, or
	// a sale's average cost could not be worked out, so CostBasis is only the
	// part the movements stated. Selling every unit makes it known again.
	UnknownCostBasis bool `json:"unknownCostBasis"`
	// AverageCost is the cost of one unit. It is nil while the units are zero
	// or negative, the basis is unknown, or it is in more than one currency.
	AverageCost *CostBasisAmount `json:"averageCost,omitempty"`
}

// CostBasisAmount is a cost basis in one currency.
type CostBasisAmount struct {
	CurrencyID int64           `json:"currencyId"`
	Amount     decimal.Decimal `json:"amount"`
}

type PositionsAtResult struct {
	Positions []PositionResult `json:"positions"`
}
