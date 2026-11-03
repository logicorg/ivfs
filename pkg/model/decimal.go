package model

import "github.com/shopspring/decimal"

const (
	// MoneyScale is the number of decimal places stored for every monetary
	// amount. Currency.Precision is only used when displaying an amount.
	MoneyScale int32 = 8

	// AssetQuantityScale preserves fractional-share and cryptocurrency
	// quantities without applying currency rounding rules.
	AssetQuantityScale int32 = 18

	// ExchangeRateScale is deliberately larger than MoneyScale so that FX
	// conversions do not lose precision before the resulting money amount is
	// rounded.
	ExchangeRateScale int32 = 18
)

// NewMoney returns value rounded to the storage precision used by monetary
// columns. All values in money columns should be constructed with this helper.
func NewMoney(value decimal.Decimal) decimal.Decimal {
	return value.Round(MoneyScale)
}

// NewAssetQuantity returns value rounded to the precision used by asset
// quantities.
func NewAssetQuantity(value decimal.Decimal) decimal.Decimal {
	return value.Round(AssetQuantityScale)
}

// NewExchangeRate returns value rounded to the precision used by exchange
// rates.
func NewExchangeRate(value decimal.Decimal) decimal.Decimal {
	return value.Round(ExchangeRateScale)
}
