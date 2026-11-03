package model

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

func TestNewMoneyRoundsToEightDecimalPlaces(t *testing.T) {
	t.Parallel()

	value := decimal.RequireFromString("12.123456789")

	require.Equal(t, "12.12345679", NewMoney(value).StringFixed(MoneyScale))
}

func TestNewAssetQuantityRetainsEighteenDecimalPlaces(t *testing.T) {
	t.Parallel()

	value := decimal.RequireFromString("0.1234567891234567899")

	require.Equal(t, "0.123456789123456790", NewAssetQuantity(value).StringFixed(AssetQuantityScale))
}
