package ledgertypes

import (
	"time"

	"github.com/shopspring/decimal"
)

// NegativeHolding warns that an asset is held in negative units in a financial
// account. It names the first point in time where that is so, which is the
// date of the movement that takes the running total below zero, and the units
// held right after it. Writes are never refused for it. The names are those of
// the account and the asset when the warning is made, so it reads as it is.
type NegativeHolding struct {
	FinancialAccountID   int64           `json:"financialAccountId"`
	FinancialAccountName string          `json:"financialAccountName"`
	AssetID              int64           `json:"assetId"`
	AssetName            string          `json:"assetName"`
	At                   time.Time       `json:"at"`
	Units                decimal.Decimal `json:"units"`
}
